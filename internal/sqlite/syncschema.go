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
	defer func() {
		if err = tx.Rollback(); err != nil {
			db.logger.LogAttrs(ctx, slog.LevelError, "failed to rollback transaction")
		}
	}()

	// Step 3: Remember schema.
	// Create schema against a temporary database so that we know what has changed.
	var (
		randomID     string
		dbNameLength uint = 20
	)
	if randomID, err = random.Letters(dbNameLength); err != nil {
		return errors.Wrap(err, "generate random ID")
	}
	schemaTargetDataSourceName := fmt.Sprintf("file:%s?mode=memory&cache=shared", randomID)
	schemaTargetDatabase, err := sql.Open("sqlite3", schemaTargetDataSourceName)
	if err != nil {
		return errors.Wrap(err, "open schema target database")
	}
	defer func() {
		if err = schemaTargetDatabase.Close(); err != nil {
			err = errors.Wrap(err, "close schema target database")
			db.logger.LogAttrs(ctx, slog.LevelError, "failed to close schema target database",
				errors.SlogError(err))
		}
	}()
	if _, err = schemaTargetDatabase.ExecContext(ctx, schemaDefinition); err != nil {
		return errors.Wrap(err, "migrate schema target database")
	}
	if _, err = tx.ExecContext(ctx, "ATTACH DATABASE ? AS schemaTarget",
		schemaTargetDataSourceName); err != nil {
		return errors.Wrap(err, "attach schema target database")
	}
	defer func() {
		if _, err = tx.ExecContext(ctx, "DETACH DATABASE schemaTarget"); err != nil {
			db.logger.LogAttrs(ctx, slog.LevelError, "failed to detach schema target database")
		}
	}()

	// Step 3-7 migrate tables.
	if err = db.migrateTables(ctx, tx); err != nil {
		return errors.Wrap(err, "migrate tables")
	}

	// Step 8: Recreate indexes and triggers associated with table if needed.
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

	return nil
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
	var changedTables []changedTable
	if changedTables, err = db.queryChangedTables(ctx, tx); err != nil {
		return errors.Wrap(err, "query changed tables")
	}

	for _, table := range changedTables {
		db.logger.LogAttrs(ctx, slog.LevelInfo, "migrating table",
			slog.String("table", table.name),
			slog.String("current_sql", table.currentSQL),
			slog.String("new_sql", table.newSQL))

		// Step 4: Create tables according to new schema on temporary names.
		tempName := table.name + "_migration_temp"
		tempNameSQL := strings.Replace(table.newSQL, table.name, tempName, 1)
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
		if _, err = tx.ExecContext(ctx, fmt.Sprintf("DROP TABLE %s;", table.name)); err != nil {
			return errors.Wrap(err, "drop old table")
		}

		// Step 7: Rename new table to old table's name.
		if _, err = tx.ExecContext(ctx, fmt.Sprintf("ALTER TABLE %s RENAME TO %s;", tempName, table.name)); err != nil {
			return errors.Wrap(err, "rename new table")
		}
	}
	return nil
}

// queryDeletedTables returns a list of tables that are present in the current schema but not in the target schema.
func (db *Database) queryDeletedTables(ctx context.Context, tx *sql.Tx) ([]string, error) {
	var (
		deletedTables []string
		err           error
	)
	if deletedTables, err = db.queryStringSlice(ctx, tx, `SELECT current.name AS deleted_table
FROM sqlite_schema AS current
LEFT JOIN schemaTarget.sqlite_schema AS target ON current.name=target.name AND current.type=target.type
WHERE current.type = 'table' AND target.type IS NULL AND current.name NOT LIKE 'sqlite_%';`); err != nil {
		return nil, errors.Wrap(err, "query string slice")
	}
	return deletedTables, nil
}

// queryNewTableSQLs returns a list of SQL statements to create new tables that are present in the target schema but not
// in the current schema.
func (db *Database) queryNewTableSQLs(ctx context.Context, tx *sql.Tx) ([]string, error) {
	var (
		newTableSQLs []string
		err          error
	)
	if newTableSQLs, err = db.queryStringSlice(ctx, tx, `SELECT target.sql AS sql
FROM sqlite_schema AS current
         RIGHT JOIN schemaTarget.sqlite_schema AS target ON CURRENT.name=target.name AND CURRENT.type=target.type
WHERE target.type = 'table' AND CURRENT.type IS NULL AND target.name NOT LIKE 'sqlite_%';`); err != nil {
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
FROM PRAGMA_TABLE_INFO(:table_name) AS current
JOIN PRAGMA_TABLE_INFO(:table_name, 'schemaTarget') AS target ON target.name = current.name;`,
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

type changedTable struct {
	name       string
	currentSQL string
	newSQL     string
}

// queryChangedTables returns a list of tables that have different schema in the current schema and the target schema.
func (db *Database) queryChangedTables(ctx context.Context, tx *sql.Tx) ([]changedTable, error) {
	var (
		changedTables []changedTable
		rows          *sql.Rows
		err           error
	)
	if rows, err = tx.QueryContext(ctx, `SELECT
    current.name AS changed_table,
    current.sql AS current_sql,
    target.sql AS new_sql
FROM sqlite_schema AS current
         JOIN schemaTarget.sqlite_schema AS target ON current.name=target.name AND current.type=target.type
WHERE current.type = 'table' AND current.name NOT LIKE 'sqlite_%' AND current.sql <> target.sql;
`); err != nil {
		return nil, errors.Wrap(err, "query")
	}
	defer func() {
		if err = rows.Close(); err != nil {
			err = errors.Wrap(err, "close rows")
			db.logger.Error("could not close rows", errors.SlogError(err))
		}
	}()
	for rows.Next() {
		var result changedTable
		if err = rows.Scan(&result.name, &result.currentSQL, &result.newSQL); err != nil {
			return nil, errors.Wrap(err, "scan table")
		}
		changedTables = append(changedTables, result)
	}
	if err = rows.Err(); err != nil {
		return nil, errors.Wrap(err, "rows error")
	}
	return changedTables, nil
}
