package main

import (
	"context"
	"github.com/myrjola/sheerluck/internal/errors"
	"github.com/myrjola/sheerluck/internal/logging"
	"github.com/myrjola/sheerluck/internal/sqlite"
	"log/slog"
	"os"
	"time"
)

func main() {
	loggerHandler := logging.NewContextHandler(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource:   false,
		Level:       slog.LevelDebug,
		ReplaceAttr: nil,
	}))
	logger := slog.New(loggerHandler)
	var (
		err       error
		start     = time.Now()
		ctx       context.Context
		sqliteURL string
		ok        bool
		cancel    context.CancelFunc
	)
	ctx = context.Background()
	ctx, cancel = context.WithTimeout(ctx, 5*time.Second) //nolint:mnd // 5 seconds

	if sqliteURL, ok = os.LookupEnv("SHEERLUCK_SQLITE_URL"); !ok {
		logger.LogAttrs(ctx, slog.LevelError, "SHEERLUCK_SQLITE_URL not set")
		os.Exit(1)
	}

	var db *sqlite.Database
	if db, err = sqlite.NewDatabase(ctx, sqliteURL, logger); err != nil {
		logger.LogAttrs(ctx, slog.LevelError, "error creating database",
			slog.String("url", sqliteURL), errors.SlogError(err))
		os.Exit(1)
	}

	// Fetch the number of users from the database and print it out as a simple smoke test.
	row := db.ReadWrite.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`)
	var count int
	if err = row.Scan(&count); err != nil {
		logger.LogAttrs(ctx, slog.LevelError, "error fetching user count", errors.SlogError(err))
		os.Exit(1)
	}
	if count == 0 {
		logger.LogAttrs(ctx, slog.LevelError, "no users found, something is likely wrong")
		os.Exit(1)
	}
	logger.LogAttrs(ctx, slog.LevelInfo, "user count", slog.Int("count", count))

	logger.LogAttrs(ctx, slog.LevelInfo, "Migration test successful 🙌", slog.Duration("duration", time.Since(start)))
	cancel()
	os.Exit(0)
}
