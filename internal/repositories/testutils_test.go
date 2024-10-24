package repositories

import (
	_ "embed"
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/myrjola/sheerluck/db"
	"os"
	"testing"
)

//go:embed testdata/fixtures.sql
var testFixtures string

// newTestDB creates a new in-memory database for testing purposes.
func newTestDB(t *testing.T) (*sqlx.DB, *sqlx.DB) {
	var (
		readWriteDB, readDB *sqlx.DB
		err                 error
	)

	if readWriteDB, readDB, err = db.NewDB(":memory:"); err != nil {
		t.Fatal(err)
	}

	// Set database to read-only mode.
	// The mode=ro flag doesn't seem to work with :memory: and cache=shared.
	readDB.MustExec("PRAGMA query_only = TRUE;")

	// Add test data
	readWriteDB.MustExec(testFixtures)

	t.Cleanup(func() {
		defer func() {
			if err := readWriteDB.Close(); err != nil {
				t.Fatal(err)
			}
			if err := readDB.Close(); err != nil {
				t.Fatal(err)
			}
		}()
	})

	return readWriteDB, readDB
}

// newBenchmarkDB creates database connection pools for benchmarking purposes.
func newBenchmarkDB(b *testing.B) (*sqlx.DB, *sqlx.DB) {
	var (
		readWriteDB, readDB *sqlx.DB
		err                 error
		benchmarkDBPath     = "./benchmark.sqlite"
	)

	if readWriteDB, readDB, err = db.NewDB(benchmarkDBPath); err != nil {
		b.Fatal(err)
	}

	b.Cleanup(func() {
		if err = readWriteDB.Close(); err != nil {
			b.Fatal(err)
		}
		if err = readDB.Close(); err != nil {
			b.Fatal(err)
		}
		_ = os.Remove(benchmarkDBPath)
		_ = os.Remove(fmt.Sprintf("%s-shm", benchmarkDBPath))
		_ = os.Remove(fmt.Sprintf("%s-wal", benchmarkDBPath))
	})

	return readWriteDB, readDB
}
