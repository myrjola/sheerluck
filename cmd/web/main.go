package main

import (
	"context"
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
	"net/http"
	"os"
	"os/signal"
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
	// FQDN is the fully qualified domain name of the server used for WebAuthn Relying Party configuration.
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

	var htmlTemplatePath string
	if htmlTemplatePath, err = resolveAndVerifyTemplatePath(cfg.TemplatePath); err != nil {
		return errors.Wrap(err, "resolve template path")
	}

	dbs, err := db.NewDB(cfg.SqliteURL)
	if err != nil {
		return errors.Wrap(err, "open db", slog.String("url", cfg.SqliteURL))
	}
	logger.LogAttrs(ctx, slog.LevelInfo, "connected to db")

	// Start goroutine that runs optimize once per hour according to suggestion at
	// https://www.sqlite.org/pragma.html#pragma_optimize.
	go func(ctx context.Context) {
		for {
			start := time.Now()
			if _, err = dbs.ReadWriteDB.Exec("PRAGMA optimize;"); err != nil {
				err = errors.Wrap(err, "optimize database")
				logger.LogAttrs(ctx, slog.LevelError, "failed to optimize database", errors.SlogError(err))
			} else {
				logger.LogAttrs(ctx, slog.LevelInfo, "optimized database",
					slog.Duration("duration", time.Since(start)))
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Hour):
				continue
			}
		}
	}(ctx)

	sessionManager := initializeSessionManager(dbs)

	var webAuthnHandler *webauthnhandler.WebAuthnHandler
	if webAuthnHandler, err = webauthnhandler.New(cfg.Addr, cfg.FQDN, logger, sessionManager, dbs); err != nil {
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

	if err = app.configureAndStartServer(ctx, cfg.Addr); err != nil {
		return errors.Wrap(err, "start server")
	}
	return nil
}

func initializeSessionManager(dbs *db.DBs) *scs.SessionManager {
	sessionManager := scs.New()
	sessionManager.Store = sqlite3store.NewWithCleanupInterval(dbs.ReadWriteDB, 24*time.Hour) //nolint:mnd // day
	sessionManager.Lifetime = 12 * time.Hour                                                  //nolint:mnd // half a day
	sessionManager.Cookie.Persist = true
	sessionManager.Cookie.Secure = true
	sessionManager.Cookie.HttpOnly = true
	sessionManager.Cookie.SameSite = http.SameSiteStrictMode
	return sessionManager
}

func main() {
	ctx := context.Background()
	loggerHandler := logging.NewContextHandler(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
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
