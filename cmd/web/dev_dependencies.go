//go:build dev
// +build dev

package main

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/myrjola/sheerluck/internal/util"
)

func init() {
	var (
		ctx  = context.Background()
		conn *pgx.Conn
	)
	c, err := util.CreateTestDB(ctx)

	connStr, err := c.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		panic(err)
	}

	// check the connection to the database
	if conn, err = pgx.Connect(ctx, connStr); err != nil {
		panic(err)
	}
	if err = conn.Close(ctx); err != nil {
		panic(err)
	}

	pgConnStr = connStr
}
