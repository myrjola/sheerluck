package main

import (
	"flag"
	"fmt"
	"github.com/alexedwards/scs/sqlite3store"
	"github.com/alexedwards/scs/v2"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/joho/godotenv"
	"github.com/myrjola/sheerluck/internal/ai"
	"github.com/myrjola/sheerluck/internal/pprofserver"
	"github.com/myrjola/sheerluck/internal/repositories"
	"github.com/myrjola/sheerluck/sqlite"
	"log/slog"
	"net/http"
	"os"
	"time"
)

type application struct {
	logger         *slog.Logger
	aiClient       ai.Client
	webAuthn       *webauthn.WebAuthn
	sessionManager *scs.SessionManager
	users          *repositories.UserRepository
}

type configuration struct {
	Addr      string `env:"SHEERLUCK_ADDR" default:"localhost:4000"`
	FQDN      string `env:"SHEERLUCK_FQDN" default:"localhost"`
	PprofPort string `env:"SHEERLUCK_PPROF_PORT" default:":6060"`
	SqliteURL string `env:"SHEERLUCK_SQLITE_URL" default:"./sheerluck.sqlite"`
}

func main() {
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

	defaultAddr := os.Getenv("SHEERLUCK_ADDR")
	if len(defaultAddr) == 0 {
		defaultAddr = "localhost:4000"
	}
	defaultFQDN := os.Getenv("SHEERLUCK_FQDN")
	if len(defaultFQDN) == 0 {
		defaultFQDN = "localhost"
	}
	defaultPprofPort := os.Getenv("SHEERLUCK_PPROF_PORT")
	if len(defaultPprofPort) == 0 {
		defaultPprofPort = ":6060"
	}
	defaultSqliteURL := os.Getenv("SHEERLUCK_SQLITE_URL")
	if len(defaultSqliteURL) == 0 {
		defaultSqliteURL = "./sheerluck.sqlite"
	}

	addr := flag.String("addr", defaultAddr, "HTTP network address")
	fqdn := flag.String("fqdn", defaultFQDN, "Fully qualified domain name for setting up Webauthn")
	pprofPort := flag.String("pprof-port", defaultPprofPort, "Port for pprof listening on localhost")
	sqliteURL := flag.String("sqlite-url", defaultSqliteURL, "SQLite URL")
	flag.Parse()

	// Initialise pprof listening on localhost so that it's not open to the world
	pprofserver.Launch(*pprofPort, logger)

	rpOrigins := []string{fmt.Sprintf("http://%s", *addr)}
	if *fqdn != "localhost" {
		rpOrigins = []string{fmt.Sprintf("https://%s", *fqdn)}
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

	db, err := sqlite.NewDB(*sqliteURL)
	if err != nil {
		logger.Error("open database %s: %w", *sqliteURL, err)
		os.Exit(1)
	}

	logger.Info("connected to db")

	sessionManager := scs.New()
	sessionManager.Store = sqlite3store.NewWithCleanupInterval(db.DB, 24*time.Hour)
	sessionManager.Lifetime = 12 * time.Hour

	users := repositories.NewUserRepository(db, logger)

	app := application{
		logger:         logger,
		aiClient:       ai.NewClient(),
		webAuthn:       webAuthn,
		sessionManager: sessionManager,
		users:          users,
	}

	logger.Info("starting server", slog.Any("addr", *addr))

	err = http.ListenAndServe(*addr, app.routes())
	logger.Error(err.Error())
	os.Exit(1)
}
