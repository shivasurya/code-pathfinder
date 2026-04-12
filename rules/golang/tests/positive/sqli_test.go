// SQL injection positive test cases — all SHOULD be detected
package main

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jmoiron/sqlx"
	"gorm.io/gorm"
)

// GO-SEC-001: database/sql injection

func sqlInjectionStdlib(w http.ResponseWriter, r *http.Request) {
	db, _ := sql.Open("postgres", "")
	id := r.FormValue("id")                                     // source
	db.Query("SELECT * FROM users WHERE id = " + id)           // SINK: SQL injection
}

func sqlInjectionStdlibExec(w http.ResponseWriter, r *http.Request) {
	db, _ := sql.Open("postgres", "")
	name := r.FormValue("name")                                 // source
	db.Exec("DELETE FROM users WHERE name = '" + name + "'")   // SINK
}

func sqlInjectionViaGinParam(db *sql.DB, c *gin.Context) {
	id := c.Param("id")                                         // source: Gin URL param
	db.QueryRow("SELECT * FROM items WHERE id = " + id)        // SINK
}

// GO-SQLI-002: pgx SQL injection — non-chained calls

func pgxSQLInjectionExec(ctx context.Context, conn *pgx.Conn, r *http.Request) {
	id := r.FormValue("id")                                     // source
	conn.Exec(ctx, "SELECT * FROM users WHERE id = "+id)       // SINK: pgx injection
}

func pgxSQLInjectionQuery(ctx context.Context, conn *pgx.Conn, c *gin.Context) {
	filter := c.Query("filter")                                  // source
	conn.Query(ctx, "SELECT * FROM logs WHERE "+filter)         // SINK: filter injection
}

func pgxSQLInjectionQueryRow(ctx context.Context, conn *pgx.Conn, r *http.Request) {
	name := r.FormValue("name")                                  // source
	conn.QueryRow(ctx, "SELECT id FROM users WHERE name='"+name+"'") // SINK
}

// GO-SQLI-003: sqlx SQL injection — non-chained calls

func sqlxInjectionGet(db *sqlx.DB, r *http.Request) {
	username := r.FormValue("user")                              // source
	db.Get(nil, "SELECT * FROM users WHERE name = '"+username+"'") // SINK
}

func sqlxInjectionSelect(db *sqlx.DB, r *http.Request) {
	col := r.FormValue("col")                                    // source
	db.Select(nil, "SELECT "+col+" FROM data")                  // SINK: column injection
}

func sqlxInjectionExec(db *sqlx.DB, c *gin.Context) {
	id := c.Query("id")                                          // source
	db.Exec("DELETE FROM sessions WHERE id = " + id)            // SINK
}

// GO-GORM-SQLI-001: GORM raw SQL injection — non-chained

func gormExecInjection(db *gorm.DB, c *gin.Context) {
	id := c.Param("id")                                          // source
	db.Exec("DELETE FROM sessions WHERE user_id = " + id)      // SINK: standalone Exec
}

func gormRawWithStringConcat(db *gorm.DB, r *http.Request) {
	search := r.FormValue("search")                              // source
	query := "SELECT * FROM products WHERE name LIKE '%" + search + "%'"
	db.Raw(query)                                                // SINK: raw query
}

// GO-GORM-SQLI-002: GORM query builder injection — non-chained

func gormOrderInjection(db *gorm.DB, c *gin.Context) {
	sort := c.Query("sort")                                      // source
	db.Order(sort)                                               // SINK: ORDER BY injection
}

func gormWhereInjection(db *gorm.DB, r *http.Request) {
	filter := r.FormValue("filter")                              // source
	db.Where(filter)                                             // SINK: WHERE injection
}

func gormGroupInjection(db *gorm.DB, c *gin.Context) {
	group := c.Query("group_by")                                 // source
	db.Group(group)                                              // SINK: GROUP BY injection
}

func gormHavingInjection(db *gorm.DB, c *gin.Context) {
	having := c.Query("having")                                  // source
	db.Having(having)                                            // SINK: HAVING injection
}

func gormSelectInjection(db *gorm.DB, r *http.Request) {
	cols := r.FormValue("columns")                               // source
	db.Select(cols)                                              // SINK: SELECT column injection
}
