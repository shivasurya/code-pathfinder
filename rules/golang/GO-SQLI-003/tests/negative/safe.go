// GO-SQLI-003 negative test cases — NONE should be detected
package main

import (
	"net/http"
	"strconv"

	"github.com/jmoiron/sqlx"
)

func sqlxSafeGet(db *sqlx.DB, r *http.Request) {
	name := r.FormValue("name")
	var user struct{ ID int }
	// SAFE: positional placeholder
	db.Get(&user, "SELECT * FROM users WHERE name = $1", name)
}

func sqlxSafeNamedExec(db *sqlx.DB, r *http.Request) {
	name := r.FormValue("name")
	// SAFE: named parameter
	db.NamedExec("INSERT INTO users (name) VALUES (:name)", map[string]interface{}{"name": name})
}

func sqlxSafeNumericID(db *sqlx.DB, r *http.Request) {
	raw := r.FormValue("id")
	id, err := strconv.Atoi(raw)
	if err != nil {
		return
	}
	db.QueryRow("SELECT name FROM users WHERE id = $1", id)
}
