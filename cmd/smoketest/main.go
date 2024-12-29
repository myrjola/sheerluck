package main

import (
	"context"
	"github.com/myrjola/sheerluck/internal/e2etest"
	"github.com/myrjola/sheerluck/internal/errors"
	"github.com/myrjola/sheerluck/internal/logging"
	"log/slog"
	"os"
	"time"
)

func TestAuth(client *e2etest.Client) error {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second) //nolint:mnd // 10 seconds
	defer cancel()
	var err error

	if _, err = client.Register(ctx); err != nil {
		return errors.Wrap(err, "register user")
	}
	if _, err = client.Logout(ctx); err != nil {
		return errors.Wrap(err, "logout user")
	}
	if _, err = client.Login(ctx); err != nil {
		return errors.Wrap(err, "login user")
	}
	return nil
}

func main() {
	loggerHandler := logging.NewContextHandler(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource:   false,
		Level:       slog.LevelDebug,
		ReplaceAttr: nil,
	}))
	logger := slog.New(loggerHandler)
	ctx := context.Background()

	if len(os.Args) != 2 { //nolint:mnd // we expect only hostname to be passed as argument.
		logger.LogAttrs(ctx, slog.LevelError, "usage: smoketest <hostname>")
		os.Exit(1)
	}

	var (
		hostname = os.Args[1]
		url      = "https://" + hostname
		client   *e2etest.Client
		err      error
	)
	ctx = logging.WithAttrs(ctx, slog.String("hostname", url))

	if client, err = e2etest.NewClient(url, hostname, url); err != nil {
		logger.LogAttrs(ctx, slog.LevelError, "error creating client", errors.SlogError(err))
		os.Exit(1)
	}
	if err = TestAuth(client); err != nil {
		logger.LogAttrs(ctx, slog.LevelError, "error testing auth", errors.SlogError(err))
		os.Exit(1)
	}

	logger.LogAttrs(ctx, slog.LevelInfo, "Smoke test successful 🙌")
	os.Exit(0)
}
