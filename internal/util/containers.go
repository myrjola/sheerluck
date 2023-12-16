package util

import (
	"context"
	"fmt"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

func CreateTestDB(ctx context.Context) (*postgres.PostgresContainer, error) {
	var (
		c   *postgres.PostgresContainer
		err error
	)

	c, err = postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:15.3-alpine"),
		postgres.WithInitScripts(filepath.Join("..", "..", "schema", "init.sql")),
		postgres.WithDatabase("sheerluck"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(5*time.Second),
			wait.ForExposedPort()),
	)
	if err != nil {
		return nil, err
	}
	// register a graceful shutdown to stop the dependencies when the application is stopped
	var gracefulStop = make(chan os.Signal, 1)
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)
	go func() {
		sig := <-gracefulStop
		fmt.Printf("caught sig: %+v\n", sig)
		err := shutdownDependencies(c)
		if err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}()
	return c, err
}

func shutdownDependencies(containers ...testcontainers.Container) error {
	ctx := context.Background()
	for _, c := range containers {
		err := c.Terminate(ctx)
		if err != nil {
			log.Println("Error terminating the backend dependency:", err)
			return err
		}
	}

	return nil
}
