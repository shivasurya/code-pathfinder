// GO-SQLI-002 positive test cases — all SHOULD be detected
package main

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

func pgxSQLInjectionExec(ctx context.Context, conn *pgx.Conn, r *http.Request) {
	id := r.FormValue("id")                                    // source
	conn.Exec(ctx, "SELECT * FROM users WHERE id = "+id)      // SINK: pgx injection
}

func pgxSQLInjectionQuery(ctx context.Context, conn *pgx.Conn, c *gin.Context) {
	filter := c.Query("filter")                                // source
	conn.Query(ctx, "SELECT * FROM logs WHERE "+filter)        // SINK
}

func pgxSQLInjectionQueryRow(ctx context.Context, conn *pgx.Conn, r *http.Request) {
	name := r.FormValue("name")                                // source
	conn.QueryRow(ctx, "SELECT id FROM users WHERE name='"+name+"'") // SINK
}
