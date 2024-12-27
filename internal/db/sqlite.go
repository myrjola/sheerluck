package db

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/myrjola/sheerluck/internal/errors"
	"github.com/myrjola/sheerluck/internal/random"
	"log/slog"
	"strings"
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
//
// The url parameter is the path to the SQLite database file or ":memory:" for an in-memory database.
func NewDB(ctx context.Context, url string) (*DBs, error) {
	var (
		err         error
		readWriteDB *sql.DB
		readDB      *sql.DB
	)

	// For in-memory databases, we need shared cache mode so that both databases access the same data.
	//
	// For parallel tests, we need to use a different database file for each test to avoid sharing data.
	// See https://www.sqlite.org/inmemorydb.html.
	isInMemory := strings.Contains(url, ":memory:")
	inMemoryConfig := ""
	if isInMemory {
		var (
			randomID     string
			dbNameLength uint = 20
		)
		if randomID, err = random.Letters(dbNameLength); err != nil {
			return nil, errors.Wrap(err, "generate random ID")
		}
		url = fmt.Sprintf("file:%s", randomID)
		inMemoryConfig = "mode=memory&cache=shared"
	}
	commonConfig := strings.Join([]string{
		// Write-ahead logging enables higher performance and concurrent readers.
		"_journal_mode=wal",
		// Avoids SQLITE_BUSY errors when database is under load.
		"_busy_timeout=5000",
		// Increases performance at the cost of durability https://www.sqlite.org/pragma.html#pragma_synchronous.
		"_synchronous=normal",
		// Enables foreign key constraints.
		"_foreign_keys=on",
		// Performance enhancement by storing temporary tables indices in memory instead of files.
		"_temp_store=memory",
		// Performance enhancement for reducing syscalls by having the pages in memory-mapped I/O.
		"_mmap_size=30000000000",
		// Recommended performance enhancement for long-lived connections.
		// See https://www.sqlite.org/pragma.html#pragma_optimize.
		"_optimize=0x10002",
	}, "&")

	// The options prefixed with underscore '_' are SQLite pragmas documented at https://www.sqlite.org/pragma.html.
	// The options without leading underscore are SQLite URI parameters documented at https://www.sqlite.org/uri.html.
	readConfig := fmt.Sprintf("file:%s?mode=ro&_txlock=deferred&_query_only=true&%s&%s", url, commonConfig, inMemoryConfig)
	readWriteConfig := fmt.Sprintf("file:%s?mode=rwc&_txlock=immediate&%s&%s", url, commonConfig, inMemoryConfig)

	if readWriteDB, err = sql.Open("sqlite3", readWriteConfig); err != nil {
		return nil, errors.Wrap(err, "open read-write database")
	}

	readWriteDB.SetMaxOpenConns(1)
	readWriteDB.SetMaxIdleConns(1)
	readWriteDB.SetConnMaxLifetime(time.Hour)
	readWriteDB.SetConnMaxIdleTime(time.Hour)

	// Initialize the database schema
	if _, err = readWriteDB.ExecContext(ctx, initialiseSchemaScript); err != nil {
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

// Runs optimize once per hour according to suggestion at https://www.sqlite.org/pragma.html#pragma_optimize.
func StartDatabaseOptimizer(ctx context.Context, dbs *DBs, logger *slog.Logger) {
	for {
		start := time.Now()
		if _, err := dbs.ReadWriteDB.ExecContext(ctx, "PRAGMA optimize;"); err != nil {
			err = errors.Wrap(err, "optimize database")
			logger.LogAttrs(ctx, slog.LevelError, "failed to optimize database", errors.SlogError(err))
		} else {
			logger.LogAttrs(ctx, slog.LevelInfo, "optimized database",
				slog.Duration("duration", time.Since(start)))
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Hour):
			continue
		}
	}
}
