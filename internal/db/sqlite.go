package db

import (
	"database/sql"
	"fmt"
	"github.com/myrjola/sheerluck/internal/errors"
	"time"

	_ "embed"
	_ "github.com/mattn/go-sqlite3" // Enable sqlite3 driver
)

//go:embed init.sql
var initialiseSchemaScript string

type DBs struct {
	ReadWriteDB *sql.DB
	ReadDB      *sql.DB
}

// NewDB establishes two database connections, one for read/write operations and one for read-only operations.
// This is a best practice mentioned in https://github.com/mattn/go-sqlite3/issues/1179#issuecomment-1638083995
func NewDB(url string) (*DBs, error) {
	var (
		err         error
		readWriteDB *sql.DB
		readDB      *sql.DB
	)

	// For in-memory databases, we need shared cache mode so that both databases access the same data.
	isInMemory := url == ":memory:"
	cacheConfig := "cache=private"
	if isInMemory {
		cacheConfig = "cache=shared"
	}
	commonConfig := "_journal_mode=wal&_busy_timeout=5000&_synchronous=normal&_foreign_keys=on"
	readConfig := fmt.Sprintf("file:%s?mode=ro&_txlock=deferred&%s&%s", url, commonConfig, cacheConfig)
	readWriteConfig := fmt.Sprintf("file:%s?mode=rwc&_txlock=immediate&%s&%s", url, commonConfig, cacheConfig)

	if readWriteDB, err = sql.Open("sqlite3", readWriteConfig); err != nil {
		return nil, errors.Wrap(err, "open read-write database")
	}

	readWriteDB.SetMaxOpenConns(1)
	readWriteDB.SetMaxIdleConns(1)
	readWriteDB.SetConnMaxLifetime(time.Hour)
	readWriteDB.SetConnMaxIdleTime(time.Hour)

	// Initialize the database schema
	if _, err = readWriteDB.Exec(initialiseSchemaScript); err != nil {
		return nil, errors.Wrap(err, "initialize schema")
	}

	if readDB, err = sql.Open("sqlite3", readConfig); err != nil {
		return nil, errors.Wrap(err, "open read database")
	}

	maxReadConns := 10
	readDB.SetMaxOpenConns(maxReadConns)
	readDB.SetMaxIdleConns(maxReadConns)
	readDB.SetConnMaxLifetime(time.Hour)
	readDB.SetConnMaxIdleTime(time.Hour)

	return &DBs{
		ReadWriteDB: readWriteDB,
		ReadDB:      readDB,
	}, nil
}
