package main

import (
	"flag"
	"fmt"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/joho/godotenv"
	"github.com/myrjola/sheerluck/internal/ai"
	"log/slog"
	"net/http"
	"os"
)

type application struct {
	logger   *slog.Logger
	aiClient ai.Client
	webAuthn *webauthn.WebAuthn
}

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

	app := application{
		logger:   logger,
		aiClient: ai.NewClient(),
		webAuthn: webAuthn,
	}

	logger.Info("starting server", slog.Any("addr", ":4000"))

	err = http.ListenAndServe(*addr, app.routes())
	logger.Error(err.Error())
	os.Exit(1)
}
