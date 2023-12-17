package sqlite

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"

	_ "embed"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed init.sql
var initialiseSchemaScript string

func NewDB(url string) (*sqlx.DB, error) {
	var (
		err error
		db  *sqlx.DB
		ctx = context.Background()
	)
	if db, err = sqlx.ConnectContext(ctx, "sqlite3", url); err != nil {
		return nil, err
	}

	// Apply recommended settings
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)
	db.SetConnMaxIdleTime(time.Hour)
	db.MustExec(`
		PRAGMA journal_mode = WAL;
		PRAGMA busy_timeout = 5000;
		PRAGMA foreign_keys = ON;
	`)

	// Initialize the database schema
	db.MustExec(initialiseSchemaScript)

	return db, nil
}
