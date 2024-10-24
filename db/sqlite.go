package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	_ "embed"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed init.sql
var initialiseSchemaScript string

// NewDB establishes two database connections, one for read/write operations and one for read-only operations.
// This is a best practice mentioned in https://github.com/mattn/go-sqlite3/issues/1179#issuecomment-1638083995
func NewDB(url string) (*sqlx.DB, *sqlx.DB, error) {
	var (
		err         error
		readWriteDB *sqlx.DB
		readDB      *sqlx.DB
		ctx         = context.Background()
	)

	// For in-memory databases, we need shared cache mode so that both databases access the same data.
	isInMemory := url == ":memory:"
	cacheConfig := "&cache=private"
	if isInMemory {
		cacheConfig = "&cache=shared"
	}
	readConfig := fmt.Sprintf("file:%s?mode=ro&_txlock=deferred&_journal_mode=wal&_busy_timeout=5000&_synchronous=normal%s", url, cacheConfig)
	readWriteConfig := fmt.Sprintf("file:%s?mode=rwc&_txlock=immediate&_journal_mode=wal&_busy_timeout=5000&_synchronous=normal%s", url, cacheConfig)

	if readWriteDB, err = sqlx.ConnectContext(ctx, "sqlite3", readWriteConfig); err != nil {
		return nil, nil, err
	}

	readWriteDB.SetMaxOpenConns(1)
	readWriteDB.SetMaxIdleConns(1)
	readWriteDB.SetConnMaxLifetime(time.Hour)
	readWriteDB.SetConnMaxIdleTime(time.Hour)

	// Initialize the database schema
	readWriteDB.MustExec(initialiseSchemaScript)

	if readDB, err = sqlx.ConnectContext(ctx, "sqlite3", readConfig); err != nil {
		return nil, nil, err
	}

	readDB.SetMaxOpenConns(10)
	readDB.SetMaxIdleConns(10)
	readDB.SetConnMaxLifetime(time.Hour)
	readDB.SetConnMaxIdleTime(time.Hour)

	return readWriteDB, readDB, nil
}
