package main

import (
	"flag"
	"github.com/joho/godotenv"
	"github.com/myrjola/sheerluck/internal/ai"
	"log"
	"log/slog"
	"net/http"
	"os"
)

type application struct {
	logger   *slog.Logger
	aiClient ai.Client
}

func main() {
	addr := flag.String("addr", ":4000", "HTTP network address")
	flag.Parse()

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	loggerHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	})
	logger := slog.New(loggerHandler)

	app := application{
		logger:   logger,
		aiClient: ai.NewClient(),
	}

	logger.Info("starting server", slog.Any("addr", ":4000"))

	err = http.ListenAndServe(*addr, app.routes())
	logger.Error(err.Error())
	os.Exit(1)
}
