package sqlite

import (
	"context"
	_ "embed"
	"github.com/myrjola/sheerluck/internal/errors"
)

//go:embed schema.sql
var schemaDefinition string

// syncronizeSchema ensures that the db schema matches the target schema defined in schema.sql.
//
// We employ a very simple declarative schema migration that:
//
// 1. Deletes deleted tables,
// 2. Creates new tables,
// 3. Migrates changed tables using 12-step schema migration https://www.sqlite.org/lang_altertable.html#otheralter.
//
// Inspired by https://david.rothlis.net/declarative-schema-migration-for-sqlite/
func (db *Database) synchronizeSchema(ctx context.Context) error {
	if _, err := db.ReadWrite.ExecContext(ctx, schemaDefinition); err != nil {
		return errors.Wrap(err, "sync schema")
	}
	return nil
}
