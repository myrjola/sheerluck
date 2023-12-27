package repositories

import (
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/myrjola/sheerluck/sqlite"
	"os"
	"testing"
)

// newTestDB creates a new in-memory database for testing purposes.
func newTestDB(t *testing.T) *sqlx.DB {
	var (
		readWriteDB, readDB *sqlx.DB
		err                 error
	)

	if readWriteDB, readDB, err = sqlite.NewDB(":memory:"); err != nil {
		t.Fatal(err)
	}
	if err = readDB.Close(); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		defer func() {
			err := readWriteDB.Close()
			if err != nil {
				t.Fatal(err)
			}
		}()
	})

	return readWriteDB
}

// newBenchmarkDB creates database connection pools for benchmarking purposes.
func newBenchmarkDB(b *testing.B) (*sqlx.DB, *sqlx.DB) {
	var (
		readWriteDB, readDB *sqlx.DB
		err                 error
		benchmarkDBPath     = "./benchmark.sqlite"
	)

	if readWriteDB, readDB, err = sqlite.NewDB(benchmarkDBPath); err != nil {
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
