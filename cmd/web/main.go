package main

import (
	"context"
	"fmt"
	"github.com/alexedwards/scs/sqlite3store"
	"github.com/alexedwards/scs/v2"
	"github.com/myrjola/sheerluck/internal/ai"
	"github.com/myrjola/sheerluck/internal/db"
	"github.com/myrjola/sheerluck/internal/envstruct"
	"github.com/myrjola/sheerluck/internal/errors"
	"github.com/myrjola/sheerluck/internal/logging"
	"github.com/myrjola/sheerluck/internal/pprofserver"
	"github.com/myrjola/sheerluck/internal/repositories"
	"github.com/myrjola/sheerluck/internal/webauthnhandler"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

type application struct {
	logger          *slog.Logger
	aiClient        ai.Client
	webAuthnHandler *webauthnhandler.WebAuthnHandler
	sessionManager  *scs.SessionManager
	investigations  *repositories.InvestigationRepository
	templateFS      fs.FS
}

type config struct {
	// Addr is the address to listen on.
	Addr string `env:"SHEERLUCK_ADDR" envDefault:"localhost:4000"`
	// FQDN is the fully qualified domain name of the server used for WebAuthn relying party configuration.
	FQDN string `env:"SHEERLUCK_FQDN" envDefault:"localhost"`
	// SqliteURL is the URL to the SQLite database. You can use ":memory:" for an ethereal in-memory database.
	SqliteURL string `env:"SHEERLUCK_SQLITE_URL" envDefault:"./sheerluck.sqlite3"`
	// PProfAddr is the optional address to listen on for the pprof server.
	PProfAddr string `env:"SHEERLUCK_PPROF_ADDR" envDefault:""`
	// TemplatePath is the path to the directory containing the HTML templates.
	TemplatePath string `env:"SHEERLUCK_TEMPLATE_PATH" envDefault:""`
}

func run(ctx context.Context, logger *slog.Logger, lookupEnv func(string) (string, bool)) error {
	var (
		cancel context.CancelFunc
		err    error
	)

	ctx, cancel = signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	var cfg config
	if err = envstruct.Populate(&cfg, lookupEnv); err != nil {
		return errors.Wrap(err, "populate config")
	}

	if cfg.PProfAddr != "" {
		pprofserver.Launch(ctx, cfg.PProfAddr, logger)
	}

	var htmlTemplatePath = cfg.TemplatePath
	if htmlTemplatePath == "" {
		// findModuleDir locates the directory containing the go.mod file.
		findModuleDir := func() (string, error) {
			var dir string
			dir, err = os.Getwd()
			if err != nil {
				return "", errors.Wrap(err, "get working directory")
			}

			for {
				if _, err = os.Stat(filepath.Join(dir, "go.mod")); err == nil {
					return dir, nil
				}

				parentDir := filepath.Dir(dir)
				if parentDir == dir { // If we reached the root directory
					break
				}
				dir = parentDir
			}

			return "", os.ErrNotExist
		}
		var modulePath string
		if modulePath, err = findModuleDir(); err != nil {
			return errors.Wrap(err, "find module dir")
		}
		htmlTemplatePath = filepath.Join(modulePath, "ui", "templates")
	}
	// check that TemplatePath exists
	var stat os.FileInfo
	if stat, err = os.Stat(htmlTemplatePath); err != nil {
		return errors.Wrap(err, "template path not found", slog.String("path", htmlTemplatePath))
	}
	if !stat.IsDir() {
		return errors.New("template path is not a directory", slog.String("path", htmlTemplatePath))
	}

	var addr = cfg.Addr
	rpOrigins := []string{fmt.Sprintf("http://%s", cfg.Addr)}
	if cfg.FQDN != "localhost" {
		rpOrigins = []string{fmt.Sprintf("https://%s", cfg.FQDN)}
	}
	dbs, err := db.NewDB(cfg.SqliteURL)
	if err != nil {
		return errors.Wrap(err, "open db", slog.String("url", cfg.SqliteURL))
	}
	logger.LogAttrs(ctx, slog.LevelInfo, "connected to db")

	sessionManager := scs.New()
	sessionManager.Store = sqlite3store.NewWithCleanupInterval(dbs.ReadWriteDB, 24*time.Hour) //nolint:mnd // day
	sessionManager.Lifetime = 12 * time.Hour                                                  //nolint:mnd // half a day
	sessionManager.Cookie.Persist = true
	sessionManager.Cookie.Secure = true
	sessionManager.Cookie.HttpOnly = true
	sessionManager.Cookie.SameSite = http.SameSiteStrictMode

	var webAuthnHandler *webauthnhandler.WebAuthnHandler
	if webAuthnHandler, err = webauthnhandler.New(cfg.FQDN, rpOrigins, logger, sessionManager, dbs); err != nil {
		return errors.Wrap(err, "new webauthn handler")
	}

	investigations := repositories.NewInvestigationRepository(dbs, logger)

	app := application{
		logger:          logger,
		aiClient:        ai.NewClient(),
		webAuthnHandler: webAuthnHandler,
		sessionManager:  sessionManager,
		investigations:  investigations,
		templateFS:      os.DirFS(htmlTemplatePath),
	}

	idleTimeout := time.Minute
	defaultTimeout := 5 * time.Second //nolint:mnd // 5 seconds should be enough even for slow LLM APIs.
	srv := &http.Server{
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
		Handler:           timeoutHandler(app.routes(), defaultTimeout),
		IdleTimeout:       idleTimeout,
		ReadTimeout:       defaultTimeout,
		WriteTimeout:      defaultTimeout,
		ReadHeaderTimeout: time.Second,
	}
	shutdownComplete := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)

		signal.Notify(sigint, os.Interrupt)
		signal.Notify(sigint, syscall.SIGTERM)

		<-sigint
		logger.LogAttrs(ctx, slog.LevelInfo, "shutting down server")

		// We received an interrupt signal, shut down.
		var shutdownContext context.Context
		shutdownContext, cancel = context.WithTimeout(context.Background(), defaultTimeout)
		defer cancel()
		if err = srv.Shutdown(shutdownContext); err != nil {
			err = errors.Wrap(err, "shutdown server")
			logger.LogAttrs(ctx, slog.LevelError, "error shutting down server", errors.SlogError(err))
		}
		close(shutdownComplete)
	}()

	var listener net.Listener
	if listener, err = net.Listen("tcp", addr); err != nil {
		return errors.Wrap(err, "TCP listen")
	}
	logger.LogAttrs(ctx, slog.LevelInfo, "starting server", slog.Any("Addr", listener.Addr().String()))
	if err = srv.Serve(listener); !errors.Is(err, http.ErrServerClosed) {
		return errors.Wrap(err, "server serve")
	}
	<-shutdownComplete

	return nil
}

func main() {
	ctx := context.Background()
	loggerHandler := logging.NewContextHandler(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource:   false,
		Level:       slog.LevelDebug,
		ReplaceAttr: nil,
	}))
	logger := slog.New(loggerHandler)
	if err := run(ctx, logger, os.LookupEnv); err != nil {
		logger.LogAttrs(ctx, slog.LevelError, "failure starting application", errors.SlogError(err))
		os.Exit(1)
	}
}
