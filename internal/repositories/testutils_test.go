package repositories

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/myrjola/sheerluck/internal/util"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"os"
	"testing"
)

func newTestDB(t *testing.T) *pgxpool.Pool {
	var (
		db      *pgxpool.Pool
		err     error
		c       *postgres.PostgresContainer
		connStr string
		ctx     = context.Background()
	)

	if c, err = util.CreateTestDB(ctx); err != nil {
		t.Fatal(err)
	}

	if connStr, err = c.ConnectionString(ctx, "sslmode=disable"); err != nil {
		t.Fatal(err)
	}

	if db, err = pgxpool.New(ctx, connStr); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		defer db.Close()

		script, err := os.ReadFile("./testdata/teardown.sql")
		if err != nil {
			t.Fatal(err)
		}
		_, err = db.Exec(ctx, string(script))
		if err != nil {
			t.Fatal(err)
		}
	})

	return db
}
