package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/alexedwards/scs/sqlite3store"
	"github.com/alexedwards/scs/v2"
	"github.com/donseba/go-htmx"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/myrjola/sheerluck/db"
	"github.com/myrjola/sheerluck/internal/ai"
	"github.com/myrjola/sheerluck/internal/broker"
	"github.com/myrjola/sheerluck/internal/pprofserver"
	"github.com/myrjola/sheerluck/internal/repositories"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type application struct {
	logger         *slog.Logger
	aiClient       ai.Client
	webAuthn       *webauthn.WebAuthn
	sessionManager *scs.SessionManager
	users          *repositories.UserRepository
	investigations *repositories.InvestigationRepository
	htmx           *htmx.HTMX
	queries        *db.Queries
	broker         *broker.ChannelBroker[uuid.UUID, struct {
		string
		error
	}]
}

func run(ctx context.Context, w io.Writer, args []string, getenv func(string) string) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	loggerHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	})
	logger := slog.New(loggerHandler)

	err := godotenv.Load()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
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
	proxyPort := flagSet.String("proxyport", "", "Proxy port for configuring webauthn in dev environment")
	if err = flagSet.Parse(args[1:]); err != nil {
		return err
	}

	// Initialise pprof listening on localhost so that it's not open to the world
	pprofserver.Launch(*pprofPort, logger)

	rpOrigins := []string{fmt.Sprintf("http://%s", *addr)}
	if *fqdn != "localhost" {
		rpOrigins = []string{fmt.Sprintf("https://%s", *fqdn)}
	}
	if *proxyPort != "" {
		rpOrigins = []string{fmt.Sprintf("http://%s:%s", *fqdn, *proxyPort)}
	}

	var webauthnConfig = &webauthn.Config{
		RPDisplayName: "Sheerluck",
		RPID:          *fqdn,
		RPOrigins:     rpOrigins,
	}

	var webAuthn *webauthn.WebAuthn
	if webAuthn, err = webauthn.New(webauthnConfig); err != nil {
		logger.Error("webauthn: %w", err)
		os.Exit(1)
	}

	readWriteDB, readDB, err := db.NewDB(*sqliteURL)
	if err != nil {
		logger.Error("open database %s: %w", *sqliteURL, err)
		os.Exit(1)
	}

	logger.Info("connected to db")

	sessionManager := scs.New()
	sessionManager.Store = sqlite3store.NewWithCleanupInterval(readWriteDB.DB, 24*time.Hour)
	sessionManager.Lifetime = 12 * time.Hour

	users := repositories.NewUserRepository(readWriteDB, readDB, logger)
	investigations := repositories.NewInvestigationRepository(readWriteDB, readDB, logger)

	channelBroker := broker.NewChannelBroker[uuid.UUID, struct {
		string
		error
	}]()

	app := application{
		logger:         logger,
		aiClient:       ai.NewClient(),
		webAuthn:       webAuthn,
		sessionManager: sessionManager,
		users:          users,
		investigations: investigations,
		htmx:           htmx.New(),
		queries:        db.New(readDB),
		broker:         channelBroker,
	}

	go func() {
		defer func() {
			if err := recover(); err != nil {
				// Safest to gracefully shut down the server in case of a panic
				app.logger.Error("channel broker: %w", err)
				if err = syscall.Kill(syscall.Getpid(), syscall.SIGINT); err != nil {
					panic(err)
				}
			}
		}()

		channelBroker.Start()
	}()

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
		logger.Error("HTTP server ListenAndServe: %v", err)
		os.Exit(1)
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
