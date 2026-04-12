// GO-SQLI-002 negative test cases — NONE should be detected
package main

import (
	"context"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"
)

func pgxSafeParamQuery(ctx context.Context, conn *pgx.Conn, r *http.Request) {
	id := r.FormValue("id")
	// SAFE: positional parameter $1 prevents injection
	conn.QueryRow(ctx, "SELECT name FROM users WHERE id = $1", id)
}

func pgxSafeNumericID(ctx context.Context, conn *pgx.Conn, r *http.Request) {
	raw := r.FormValue("id")
	id, err := strconv.Atoi(raw) // sanitizer
	if err != nil {
		return
	}
	conn.QueryRow(ctx, "SELECT name FROM users WHERE id = $1", id)
}

func pgxSafeConstant(ctx context.Context, conn *pgx.Conn) {
	// SAFE: no user input
	conn.Query(ctx, "SELECT * FROM users WHERE active = true")
}
