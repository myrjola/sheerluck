package repositories_test

import (
	"context"
	_ "embed"
	"fmt"
	"github.com/myrjola/sheerluck/internal/db"
	"os"
	"testing"
)

//go:embed testdata/fixtures.sql
var testFixtures string

// newTestDB creates a new in-memory database for testing purposes.
func newTestDB(t *testing.T) *db.DBs {
	var (
		dbs *db.DBs
		err error
	)

	if dbs, err = db.NewDB(context.Background(), ":memory:"); err != nil {
		t.Fatal(err)
	}

	// Add test data
	if _, err = dbs.ReadWriteDB.Exec(testFixtures); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		defer func() {
			if err = dbs.ReadWriteDB.Close(); err != nil {
				t.Fatal(err)
			}
			if err = dbs.ReadDB.Close(); err != nil {
				t.Fatal(err)
			}
		}()
	})

	return dbs
}

// newBenchmarkDB creates database connection pools for benchmarking purposes.
func newBenchmarkDB(b *testing.B) *db.DBs {
	var (
		dbs             *db.DBs
		err             error
		benchmarkDBPath = "./benchmark.sqlite"
	)

	if dbs, err = db.NewDB(context.Background(), benchmarkDBPath); err != nil {
		b.Fatal(err)
	}

	b.Cleanup(func() {
		if err = dbs.ReadWriteDB.Close(); err != nil {
			b.Fatal(err)
		}
		if err = dbs.ReadDB.Close(); err != nil {
			b.Fatal(err)
		}
		_ = os.Remove(benchmarkDBPath)
		_ = os.Remove(fmt.Sprintf("%s-shm", benchmarkDBPath))
		_ = os.Remove(fmt.Sprintf("%s-wal", benchmarkDBPath))
	})

	return dbs
}
