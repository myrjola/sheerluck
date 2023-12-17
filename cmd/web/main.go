package main

import (
	"flag"
	"fmt"
	"github.com/alexedwards/scs/sqlite3store"
	"github.com/alexedwards/scs/v2"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/joho/godotenv"
	"github.com/myrjola/sheerluck/internal/ai"
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

func main() {
	addr := flag.String("addr", ":4000", "HTTP network address")
	dbUrl := flag.String("sqlite-url", "./sheerluck.sqlite", "SQLite URL")
	flag.Parse()

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

	origin := "localhost"
	var wconfig = &webauthn.Config{
		RPDisplayName: "Sheerluck",
		RPID:          origin,
		RPOrigins:     []string{fmt.Sprintf("http://%s%s", origin, *addr)},
	}

	var webAuthn *webauthn.WebAuthn
	if webAuthn, err = webauthn.New(wconfig); err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	db, err := sqlite.NewDB(*dbUrl)

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
