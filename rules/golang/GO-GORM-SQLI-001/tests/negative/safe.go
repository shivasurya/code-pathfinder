// GO-GORM-SQLI-001 negative test cases — NONE should be detected
package main

import (
	"net/http"

	"gorm.io/gorm"
)

func gormSafeRawParam(db *gorm.DB, r *http.Request) {
	search := r.FormValue("search")
	// SAFE: GORM ? placeholder
	var results []struct{ Name string }
	db.Raw("SELECT * FROM products WHERE name LIKE ?", "%"+search+"%").Scan(&results)
}

func gormSafeExecParam(db *gorm.DB, r *http.Request) {
	id := r.FormValue("id")
	// SAFE: positional parameter
	db.Exec("UPDATE users SET last_seen = NOW() WHERE id = ?", id)
}

func gormSafeORM(db *gorm.DB, r *http.Request) {
	name := r.FormValue("name")
	// SAFE: GORM ORM query builder with struct conditions
	var user struct{ ID int; Name string }
	db.Where(&struct{ Name string }{Name: name}).First(&user)
}
