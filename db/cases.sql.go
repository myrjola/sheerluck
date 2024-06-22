// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0
// source: cases.sql

package db

import (
	"context"
)

const getCase = `-- name: GetCase :one
SELECT id, name, author
FROM cases
WHERE id = ?1
`

func (q *Queries) GetCase(ctx context.Context, caseid string) (Case, error) {
	row := q.db.QueryRowContext(ctx, getCase, caseid)
	var i Case
	err := row.Scan(&i.ID, &i.Name, &i.Author)
	return i, err
}

const listCases = `-- name: ListCases :many
SELECT id, name, author
FROM cases
`

func (q *Queries) ListCases(ctx context.Context) ([]Case, error) {
	rows, err := q.db.QueryContext(ctx, listCases)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Case
	for rows.Next() {
		var i Case
		if err := rows.Scan(&i.ID, &i.Name, &i.Author); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}