package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/alexedwards/scs/sqlite3store"
	"github.com/alexedwards/scs/v2"
	"github.com/joho/godotenv"
	"github.com/myrjola/sheerluck/db"
	"github.com/myrjola/sheerluck/internal/ai"
	"github.com/myrjola/sheerluck/internal/errors"
	"github.com/myrjola/sheerluck/internal/logging"
	"github.com/myrjola/sheerluck/internal/pprofserver"
	"github.com/myrjola/sheerluck/internal/repositories"
	"github.com/myrjola/sheerluck/internal/webauthnhandler"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type application struct {
	logger          *slog.Logger
	aiClient        ai.Client
	webAuthnHandler *webauthnhandler.WebAuthnHandler
	sessionManager  *scs.SessionManager
	investigations  *repositories.InvestigationRepository
}

func run(ctx context.Context, w io.Writer, args []string, getenv func(string) string) error {
	var cancel context.CancelFunc
	ctx, cancel = signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	loggerHandler := logging.NewContextHandler(slog.NewTextHandler(w, &slog.HandlerOptions{
		AddSource:   false,
		Level:       slog.LevelDebug,
		ReplaceAttr: nil,
	}))
	logger := slog.New(loggerHandler)

	err := godotenv.Load()
	if err != nil {
		return errors.Wrap(err, "load .env")
	}

	defaultAddr := getenv("SHEERLUCK_ADDR")
	if len(defaultAddr) == 0 {
		defaultAddr = "localhost:4000"
	}
	defaultFQDN := getenv("SHEERLUCK_FQDN")
	if len(defaultFQDN) == 0 {
		defaultFQDN = "localhost"
	}
	defaultPprofPort := getenv("SHEERLUCK_PPROF_ADDR")
	if len(defaultPprofPort) == 0 {
		defaultPprofPort = "localhost:6060"
	}
	defaultSqliteURL := getenv("SHEERLUCK_SQLITE_URL")
	if len(defaultSqliteURL) == 0 {
		defaultSqliteURL = "./sheerluck.sqlite"
	}

	flagSet := flag.NewFlagSet(args[0], flag.ExitOnError)
	addr := flagSet.String("addr", defaultAddr, "HTTP network address")
	fqdn := flagSet.String("fqdn", defaultFQDN, "Fully qualified domain name for setting up Webauthn")
	pprofPort := flagSet.String("pprof-addr", defaultPprofPort, "HTTP network address for pprof")
	sqliteURL := flagSet.String("sqlite-url", defaultSqliteURL, "SQLite URL")
	if err = flagSet.Parse(args[1:]); err != nil {
		return err
	}

	// Initialise pprof listening on localhost so that it's not open to the world
	pprofserver.Launch(*pprofPort, logger)

	rpOrigins := []string{fmt.Sprintf("http://%s", *addr)}
	if *fqdn != "localhost" {
		rpOrigins = []string{fmt.Sprintf("https://%s", *fqdn)}
	}
	dbs, err := db.NewDB(*sqliteURL)
	if err != nil {
		return errors.Wrap(err, "open db", slog.String("url", *sqliteURL))
	}
	logger.LogAttrs(ctx, slog.LevelInfo, "connected to db")

	sessionManager := scs.New()
	sessionManager.Store = sqlite3store.NewWithCleanupInterval(dbs.ReadWriteDB, 24*time.Hour) //nolint:mnd
	sessionManager.Lifetime = 12 * time.Hour                                                  //nolint:mnd
	sessionManager.Cookie.Persist = true
	sessionManager.Cookie.Secure = true
	sessionManager.Cookie.HttpOnly = true
	sessionManager.Cookie.SameSite = http.SameSiteStrictMode

	var webAuthnHandler *webauthnhandler.WebAuthnHandler
	if webAuthnHandler, err = webauthnhandler.New(*fqdn, rpOrigins, logger, sessionManager, dbs); err != nil {
		return errors.Wrap(err, "new webauthn handler")
	}

	investigations := repositories.NewInvestigationRepository(dbs, logger)

	app := application{
		logger:          logger,
		aiClient:        ai.NewClient(),
		webAuthnHandler: webAuthnHandler,
		sessionManager:  sessionManager,
		investigations:  investigations,
	}

	logger.Info("starting server", slog.Any("addr", *addr))

	srv := &http.Server{
		Addr:              *addr,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
		Handler:           app.routes(),
		IdleTimeout:       time.Minute,
		ReadTimeout:       time.Minute,
		WriteTimeout:      time.Minute,
		ReadHeaderTimeout: 5 * time.Second,
	}
	shutdownComplete := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)

		signal.Notify(sigint, os.Interrupt)
		signal.Notify(sigint, syscall.SIGTERM)

		<-sigint
		logger.Info("shutting down server")

		// We received an interrupt signal, shut down.
		if err := srv.Shutdown(context.Background()); err != nil {
			// Error from closing listeners, or context timeout:
			logger.Error("HTTP server shutdown: %v", err)
		}
		close(shutdownComplete)
	}()

	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		// Error starting or closing listener:
		return errors.Wrap(err, "listen and serve")
	}
	<-shutdownComplete

	return nil
}

func main() {
	ctx := context.Background()
	if err := run(ctx, os.Stdout, os.Args, os.Getenv); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
