// GO-SEC-001 positive test cases — all SHOULD be detected
package main

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
)

func sqlInjectionQuery(w http.ResponseWriter, r *http.Request) {
	db, _ := sql.Open("postgres", "")
	id := r.FormValue("id")                                   // source
	db.Query("SELECT * FROM users WHERE id = " + id)         // SINK: SQL injection
}

func sqlInjectionExec(w http.ResponseWriter, r *http.Request) {
	db, _ := sql.Open("postgres", "")
	name := r.FormValue("name")                               // source
	db.Exec("DELETE FROM users WHERE name = '" + name + "'") // SINK
}

func sqlInjectionQueryRow(db *sql.DB, c *gin.Context) {
	id := c.Param("id")                                       // source: Gin URL param
	db.QueryRow("SELECT * FROM items WHERE id = " + id)      // SINK
}

func sqlInjectionQueryContext(db *sql.DB, r *http.Request) {
	filter := r.FormValue("filter")
	db.QueryContext(nil, "SELECT * FROM logs WHERE "+filter)  // SINK
}
