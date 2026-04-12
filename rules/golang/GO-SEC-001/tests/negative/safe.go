// GO-SEC-001 negative test cases — NONE should be detected
package main

import (
	"database/sql"
	"net/http"
	"strconv"
)

func safeSQLParamQuery(db *sql.DB, r *http.Request) {
	id := r.FormValue("id")
	// SAFE: parameterized query
	db.Query("SELECT * FROM users WHERE id = $1", id)
}

func safeSQLParamExec(db *sql.DB, r *http.Request) {
	name := r.FormValue("name")
	// SAFE: placeholder prevents injection
	db.Exec("DELETE FROM users WHERE name = $1", name)
}

func safeSQLNumericSanitized(db *sql.DB, r *http.Request) {
	raw := r.FormValue("id")
	id, err := strconv.Atoi(raw) // sanitizer: parsed as integer
	if err != nil {
		return
	}
	// SAFE: id is now a typed int, cannot be a SQL fragment
	db.Query("SELECT * FROM users WHERE id = $1", id)
}

func safeSQLConstant(db *sql.DB) {
	// SAFE: no user input, constant query
	db.Query("SELECT * FROM users WHERE active = true")
}
