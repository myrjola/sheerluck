package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/alexedwards/scs/pgxstore"
	"github.com/alexedwards/scs/v2"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/myrjola/sheerluck/internal/ai"
	"github.com/myrjola/sheerluck/internal/repositories"
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

var pgConnStr = ""

func main() {
	addr := flag.String("addr", ":4000", "HTTP network address")
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

	var db *pgxpool.Pool
	ctx := context.Background()
	if db, err = pgxpool.New(ctx, pgConnStr); err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("connected to db")

	sessionManager := scs.New()
	sessionManager.Store = pgxstore.NewWithCleanupInterval(db, 24*time.Hour)
	sessionManager.Lifetime = 12 * time.Hour

	users := repositories.NewUserRepository(db, logger)

	app := application{
		logger:         logger,
		aiClient:       ai.NewClient(),
		webAuthn:       webAuthn,
		sessionManager: sessionManager,
		users:          users,
	}

	logger.Info("starting server", slog.Any("addr", ":4000"))

	err = http.ListenAndServe(*addr, app.routes())
	logger.Error(err.Error())
	os.Exit(1)
}
