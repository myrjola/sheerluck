package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/myrjola/sheerluck/internal/errors"
	"github.com/myrjola/sheerluck/internal/random"
	"log/slog"
	"os"
	"strings"
	"syscall"
	"time"
)

// migrate ensures that the db schema matches the target schema defined in schema.sql.
//
// We employ a very simple declarative schema migration that:
//
// 1. Deletes deleted tables,
// 2. Creates new tables,
// 3. Migrates changed tables using 12-step schema migration https://www.sqlite.org/lang_altertable.html#otheralter.
//
// Inspired by https://david.rothlis.net/declarative-schema-migration-for-sqlite/
func (db *Database) migrate(ctx context.Context, schemaDefinition string) error {
	var err error
	// 12-step schema migration starts here. See https://www.sqlite.org/lang_altertable.html#otheralter.
	start := time.Now()

	closeDatabase, err := db.attachSchemaTargetDatabase(ctx, schemaDefinition)
	if err != nil {
		return errors.Wrap(err, "attach schema target database")
	}
	defer closeDatabase()

	// Step 1: Disable foreign key validation temporarily.
	if _, err = db.ReadWrite.ExecContext(ctx, "PRAGMA foreign_keys = OFF"); err != nil {
		return errors.Wrap(err, "disable foreign key validation")
	}
	// Step 12: Re-enable foreign key validation.
	defer func() {
		if _, err = db.ReadWrite.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
			err = errors.Wrap(err, "re-enable foreign key validation")
			db.logger.LogAttrs(ctx, slog.LevelError, "exit to avoid data corruption", errors.SlogError(err))
			err = syscall.Kill(syscall.Getpid(), syscall.SIGINT)
			if err != nil {
				os.Exit(1)
			}
		}
	}()

	// Step 2: Start transaction.
	var tx *sql.Tx
	if tx, err = db.ReadWrite.BeginTx(ctx, nil); err != nil {
		return errors.Wrap(err, "start transaction")
	}
	defer db.rollback(ctx, tx)()

	// Step 3-7 migrate tables.
	if err = db.migrateTables(ctx, tx); err != nil {
		return errors.Wrap(err, "migrate tables")
	}

	// Step 8: Recreate indexes and triggers associated with table if needed.
	if err = db.migrateTriggers(ctx, tx); err != nil {
		return errors.Wrap(err, "migrate triggers")
	}
	// Step 9: Recreate views associated with table.
	// Step 10: Check foreign key constraints.
	if _, err = tx.ExecContext(ctx, "PRAGMA foreign_key_check"); err != nil {
		return errors.Wrap(err, "foreign key check")
	}

	// Step 11: Commit transaction from step 2.
	if err = tx.Commit(); err != nil {
		return errors.Wrap(err, "commit transaction")
	}

	// Step 12: is in defer above.

	db.logger.LogAttrs(ctx, slog.LevelInfo, "migrated tables", slog.Duration("duration", time.Since(start)))

	return nil
}

// attachSchemaTargetDatabase attaches a temporary database initialised with the target schema and returns
// a function to detach the database that must be called after the migration.
func (db *Database) attachSchemaTargetDatabase(ctx context.Context, schemaDefinition string) (func(), error) {
	// Create schema against a temporary database so that we know what has changed.
	var (
		randomID     string
		dbNameLength uint = 20
		err          error
	)
	if randomID, err = random.Letters(dbNameLength); err != nil {
		return nil, errors.Wrap(err, "generate random ID")
	}
	schemaTargetDataSourceName := fmt.Sprintf("file:%s?mode=memory&cache=shared", randomID)
	schemaTargetDatabase, err := sql.Open("sqlite3", schemaTargetDataSourceName)
	if err != nil {
		return nil, errors.Wrap(err, "open schema target database")
	}
	defer func() {
		if err = schemaTargetDatabase.Close(); err != nil {
			err = errors.Wrap(err, "close schema target database")
			db.logger.LogAttrs(ctx, slog.LevelError, "failed to close schema target database",
				errors.SlogError(err))
		}
	}()
	if _, err = schemaTargetDatabase.ExecContext(ctx, schemaDefinition); err != nil {
		return nil, errors.Wrap(err, "migrate schema target database")
	}
	if _, err = db.ReadWrite.ExecContext(ctx, "ATTACH DATABASE ? AS schemaTarget",
		schemaTargetDataSourceName); err != nil {
		return nil, errors.Wrap(err, "attach schema target database")
	}
	return func() {
		if _, err = db.ReadWrite.ExecContext(ctx, "DETACH DATABASE schemaTarget"); err != nil {
			db.logger.LogAttrs(ctx, slog.LevelError, "failed to detach schema target database", errors.SlogError(err))
		}
	}, nil
}

// rollback rolls back given transaction.
func (db *Database) rollback(ctx context.Context, tx *sql.Tx) func() {
	return func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			err = errors.Wrap(err, "rollback transaction")
			db.logger.LogAttrs(ctx, slog.LevelError, "failed to rollback transaction", errors.SlogError(err))
		}
	}
}

// migrateTables ensures table schema is synchronized between databases.
func (db *Database) migrateTables(ctx context.Context, tx *sql.Tx) error {
	// Step 3: Remember schema (also includes trivial creation and deletion of tables).
	var err error

	// Drop deleted tables.
	var deletedTables []string
	if deletedTables, err = db.queryDeletedTables(ctx, tx); err != nil {
		return errors.Wrap(err, "query deleted tables")
	}
	for _, table := range deletedTables {
		db.logger.LogAttrs(ctx, slog.LevelInfo, "dropping table", slog.String("table", table))
		if _, err = tx.ExecContext(ctx, "DROP TABLE ?;", table); err != nil {
			return errors.Wrap(err, "drop table", slog.String("table", table))
		}
	}

	// Create new tables.
	var newTableSQLs []string
	if newTableSQLs, err = db.queryNewTableSQLs(ctx, tx); err != nil {
		return errors.Wrap(err, "query new table SQLs")
	}
	for _, newTableSQL := range newTableSQLs {
		db.logger.LogAttrs(ctx, slog.LevelInfo, "creating table", slog.String("query", newTableSQL))
		if _, err = tx.ExecContext(ctx, newTableSQL); err != nil {
			return errors.Wrap(err, "create table")
		}
	}

	// Identify tables with changed schema and continue the 12-step schema migration with them.
	var changedTables []changedSchema
	if changedTables, err = db.queryChangedTables(ctx, tx); err != nil {
		return errors.Wrap(err, "query changed tables")
	}

	for _, table := range changedTables {
		db.logger.LogAttrs(ctx, slog.LevelInfo, "migrating table",
			slog.String("table", table.name),
			slog.String("live_sql", table.liveSQL),
			slog.String("new_sql", table.newSQL))

		// Step 4: Create tables according to new schema on temporary names.
		tempName := table.name + "_migration_temp"
		tempNameSQL := strings.Replace(table.newSQL, table.name, tempName, 1)
		db.logger.LogAttrs(ctx, slog.LevelInfo, "creating new table to temporary name",
			slog.String("query", tempNameSQL))
		if _, err = tx.ExecContext(ctx, tempNameSQL); err != nil {
			return errors.Wrap(err, "create new table to temporary name", slog.String("query", tempNameSQL))
		}

		// Step 5: Copy common columns between tables.
		var commonColumns []string
		if commonColumns, err = db.queryCommonColumns(ctx, tx, table.name); err != nil {
			return errors.Wrap(err, "query common columns")
		}
		common := strings.Join(commonColumns, ", ")
		copySQL := fmt.Sprintf("INSERT INTO %s (%s) SELECT %s FROM %s;", //nolint: gosec // we trust the query.
			tempName, common, common, table.name)
		db.logger.LogAttrs(ctx, slog.LevelInfo, "copying data", slog.String("query", copySQL))
		if _, err = tx.ExecContext(ctx, copySQL); err != nil {
			return errors.Wrap(err, "copy data")
		}

		// Step 6: Drop the old table.
		dropSQL := fmt.Sprintf("DROP TABLE %s;", table.name)
		db.logger.LogAttrs(ctx, slog.LevelInfo, "dropping old table", slog.String("query", dropSQL))
		if _, err = tx.ExecContext(ctx, dropSQL); err != nil {
			return errors.Wrap(err, "drop old table")
		}

		// Step 7: Rename new table to old table's name.
		renameSQL := fmt.Sprintf("ALTER TABLE %s RENAME TO %s;", tempName, table.name)
		db.logger.LogAttrs(ctx, slog.LevelInfo, "renaming new table", slog.String("query", renameSQL))
		if _, err = tx.ExecContext(ctx, renameSQL); err != nil {
			return errors.Wrap(err, "rename new table")
		}
	}
	return nil
}

// queryDeletedTables returns a list of tables that are present in the live schema but not in the target schema.
func (db *Database) queryDeletedTables(ctx context.Context, tx *sql.Tx) ([]string, error) {
	var (
		deletedTables []string
		err           error
	)
	if deletedTables, err = db.queryStringSlice(ctx, tx, `SELECT live.name AS deleted_table
FROM sqlite_schema AS live
         LEFT JOIN schemaTarget.sqlite_schema AS target ON live.name = target.name AND live.type = target.type
WHERE live.type = 'table'
  AND target.type IS NULL
  AND live.name NOT LIKE 'sqlite_%'
  AND live.name NOT LIKE '_litestream_%'`); err != nil {
		return nil, errors.Wrap(err, "query string slice")
	}
	return deletedTables, nil
}

// queryNewTableSQLs returns a list of SQL statements to create new tables that are present in the target schema but not
// in the live schema.
func (db *Database) queryNewTableSQLs(ctx context.Context, tx *sql.Tx) ([]string, error) {
	var (
		newTableSQLs []string
		err          error
	)
	if newTableSQLs, err = db.queryStringSlice(ctx, tx, `SELECT target.sql AS sql
FROM sqlite_schema AS live RIGHT JOIN schemaTarget.sqlite_schema AS target
ON live.name=target.name AND live.type=target.type
WHERE target.type = 'table'
  AND live.type IS NULL
  AND target.name NOT LIKE 'sqlite_%'
  AND target.name NOT LIKE '_litestream_%'`); err != nil {
		return nil, errors.Wrap(err, "query string slice")
	}
	return newTableSQLs, nil
}

func (db *Database) queryCommonColumns(ctx context.Context, tx *sql.Tx, table string) ([]string, error) {
	var (
		commonColumns []string
		err           error
	)
	// We wrap the column names in with double quotes to handle column names that are SQLite keywords.
	if commonColumns, err = db.queryStringSlice(ctx, tx, `SELECT '"' || target.name || '"'
FROM PRAGMA_TABLE_INFO(:table_name) AS live
JOIN PRAGMA_TABLE_INFO(:table_name, 'schemaTarget') AS target ON target.name = live.name`,
		sql.Named("table_name", table)); err != nil {
		return nil, errors.Wrap(err, "query string slice")
	}
	return commonColumns, nil
}

// queryStringSlice returns a slice of strings from a query and its args.
//
// It is used to query a single column from a table.
func (db *Database) queryStringSlice(ctx context.Context, tx *sql.Tx, query string, args ...any) ([]string, error) {
	var (
		results []string
		rows    *sql.Rows
		err     error
	)
	if rows, err = tx.QueryContext(ctx, query, args...); err != nil {
		return nil, errors.Wrap(err, "query")
	}
	defer func() {
		if err = rows.Close(); err != nil {
			err = errors.Wrap(err, "close rows")
			db.logger.Error("could not close rows", errors.SlogError(err))
		}
	}()
	for rows.Next() {
		var result string
		if err = rows.Scan(&result); err != nil {
			return nil, errors.Wrap(err, "scan table")
		}
		results = append(results, result)
	}
	if err = rows.Err(); err != nil {
		return nil, errors.Wrap(err, "rows error")
	}
	return results, nil
}

type changedSchema struct {
	name    string
	liveSQL string
	newSQL  string
}

// queryChangedSchemas returns a list of entities that have different schema in the live schema and the target schema.
func (db *Database) queryChangedSchemas(ctx context.Context, tx *sql.Tx, query string) ([]changedSchema, error) {
	var (
		changedSchemas []changedSchema
		rows           *sql.Rows
		err            error
	)
	if rows, err = tx.QueryContext(ctx, query); err != nil {
		return nil, errors.Wrap(err, "query")
	}
	defer func() {
		if err = rows.Close(); err != nil {
			err = errors.Wrap(err, "close rows")
			db.logger.Error("could not close rows", errors.SlogError(err))
		}
	}()
	for rows.Next() {
		var result changedSchema
		if err = rows.Scan(&result.name, &result.liveSQL, &result.newSQL); err != nil {
			return nil, errors.Wrap(err, "scan table")
		}
		changedSchemas = append(changedSchemas, result)
	}
	if err = rows.Err(); err != nil {
		return nil, errors.Wrap(err, "rows error")
	}
	return changedSchemas, nil
}

// queryChangedTables returns a list of tables that have different schema in the live schema and the target schema.
func (db *Database) queryChangedTables(ctx context.Context, tx *sql.Tx) ([]changedSchema, error) {
	var (
		err           error
		changedTables []changedSchema
	)
	if changedTables, err = db.queryChangedSchemas(ctx, tx, `SELECT live.name AS changed_table,
       live.sql  AS live_sql,
       target.sql   AS new_sql
FROM sqlite_schema AS live
         JOIN schemaTarget.sqlite_schema AS target ON live.name = target.name AND live.type = target.type
WHERE live.type = 'table'
  AND live.name NOT LIKE 'sqlite_%'
  AND live.name NOT LIKE '_litestream_%'
  -- The table rename operation adds double quotes around the table name, so we remove them for this diff.
  AND REPLACE(live.sql, '"', '') <> REPLACE(target.sql, '"', '')
`); err != nil {
		return nil, errors.Wrap(err, "query changed schemas")
	}
	return changedTables, nil
}

// queryChangedTriggers returns a list of triggers that have different sql in the live schema and the target schema.
func (db *Database) queryChangedTriggers(ctx context.Context, tx *sql.Tx) ([]changedSchema, error) {
	var (
		err             error
		changedTriggers []changedSchema
	)
	if changedTriggers, err = db.queryChangedSchemas(ctx, tx, `SELECT live.name  AS changed_trigger,
       live.sql   AS live_sql,
       target.sql AS new_sql
FROM sqlite_schema AS live
         JOIN schemaTarget.sqlite_schema AS target ON live.name = target.name AND live.type = target.type
WHERE live.type = 'trigger'
  AND live.name NOT LIKE 'sqlite_%'
  AND live.sql <> target.sql`); err != nil {
		return nil, errors.Wrap(err, "query changed schemas")
	}
	return changedTriggers, nil
}

// migrateTriggers ensures triggers are synchronized between databases.
func (db *Database) migrateTriggers(ctx context.Context, tx *sql.Tx) error {
	var err error

	var deleted []string
	if deleted, err = db.queryStringSlice(ctx, tx, `SELECT live.name AS deleted_trigger
FROM sqlite_schema AS live
         LEFT JOIN schemaTarget.sqlite_schema AS target ON live.name = target.name AND live.type = target.type
WHERE live.type = 'trigger'
  AND target.type IS NULL
  AND live.name NOT LIKE 'sqlite_%'`); err != nil {
		return errors.Wrap(err, "query deleted triggers")
	}
	for _, trigger := range deleted {
		db.logger.LogAttrs(ctx, slog.LevelInfo, "dropping trigger", slog.String("trigger", trigger))
		if _, err = tx.ExecContext(ctx, "DROP TRIGGER ?;", trigger); err != nil {
			return errors.Wrap(err, "drop trigger", slog.String("trigger", trigger))
		}
	}

	var created []string
	if created, err = db.queryStringSlice(ctx, tx, `SELECT target.sql AS new_trigger_sql
FROM sqlite_schema AS live
         RIGHT JOIN schemaTarget.sqlite_schema AS target ON live.name = target.name AND live.type = target.type
WHERE target.type = 'trigger'
  AND live.type IS NULL
  AND target.name NOT LIKE 'sqlite_%'`); err != nil {
		return errors.Wrap(err, "query created triggers")
	}
	for _, newTriggerSQL := range created {
		db.logger.LogAttrs(ctx, slog.LevelInfo, "creating trigger", slog.String("query", newTriggerSQL))
		if _, err = tx.ExecContext(ctx, newTriggerSQL); err != nil {
			return errors.Wrap(err, "create trigger")
		}
	}

	// Identify tables with changed schema and continue the 12-step schema migration with them.
	var changedTriggers []changedSchema
	if changedTriggers, err = db.queryChangedTriggers(ctx, tx); err != nil {
		return errors.Wrap(err, "query changed triggers")
	}

	for _, trigger := range changedTriggers {
		db.logger.LogAttrs(ctx, slog.LevelInfo, "migrating trigger",
			slog.String("trigger", trigger.name),
			slog.String("live_sql", trigger.liveSQL),
			slog.String("new_sql", trigger.newSQL))

		dropSQL := fmt.Sprintf("DROP TRIGGER %s;", trigger.name)
		db.logger.LogAttrs(ctx, slog.LevelInfo, "dropping old trigger", slog.String("query", dropSQL))
		if _, err = tx.ExecContext(ctx, dropSQL); err != nil {
			return errors.Wrap(err, "drop old trigger")
		}

		// Step 7: Rename new trigger to old trigger's name.
		db.logger.LogAttrs(ctx, slog.LevelInfo, "creating new trigger", slog.String("query", trigger.newSQL))
		if _, err = tx.ExecContext(ctx, trigger.newSQL); err != nil {
			return errors.Wrap(err, "create new trigger")
		}
	}
	return nil
}
