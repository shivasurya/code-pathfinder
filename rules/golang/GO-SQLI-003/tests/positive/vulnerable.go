// GO-SQLI-003 positive test cases — all SHOULD be detected
package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func sqlxInjectionGet(db *sqlx.DB, r *http.Request) {
	username := r.FormValue("user")                               // source
	db.Get(nil, "SELECT * FROM users WHERE name = '"+username+"'") // SINK
}

func sqlxInjectionSelect(db *sqlx.DB, r *http.Request) {
	col := r.FormValue("col")                                     // source
	db.Select(nil, "SELECT "+col+" FROM data")                    // SINK: column injection
}

func sqlxInjectionExec(db *sqlx.DB, c *gin.Context) {
	id := c.Query("id")                                           // source
	db.Exec("DELETE FROM sessions WHERE id = " + id)             // SINK
}

func sqlxInjectionQueryRow(db *sqlx.DB, r *http.Request) {
	email := r.FormValue("email")
	db.QueryRow("SELECT id FROM users WHERE email='" + email + "'") // SINK
}
