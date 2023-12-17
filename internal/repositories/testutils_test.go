package repositories

import (
	"github.com/jmoiron/sqlx"
	"github.com/myrjola/sheerluck/sqlite"
	"testing"
)

func newTestDB(t *testing.T) *sqlx.DB {
	var (
		db  *sqlx.DB
		err error
	)

	if db, err = sqlite.NewDB(":memory:"); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		defer func() {
			err := db.Close()
			if err != nil {
				t.Fatal(err)
			}
		}()
	})

	return db
}
