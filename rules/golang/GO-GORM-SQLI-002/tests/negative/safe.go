// GO-GORM-SQLI-002 negative test cases — NONE should be detected
package main

import (
	"net/http"

	"gorm.io/gorm"
)

func gormSafeOrder(db *gorm.DB, r *http.Request) {
	sort := r.FormValue("sort")
	// SAFE: allowlist validation
	if sort != "asc" && sort != "desc" {
		sort = "asc"
	}
	db.Order("created_at " + sort) // safe after validation
}

func gormSafeWhere(db *gorm.DB, r *http.Request) {
	name := r.FormValue("name")
	// SAFE: struct condition — no raw SQL injection possible
	var users []struct{ Name string }
	db.Where(&struct{ Name string }{Name: name}).Find(&users)
}

func gormSafeWhereParam(db *gorm.DB, r *http.Request) {
	name := r.FormValue("name")
	// SAFE: ? placeholder in Where
	db.Where("name = ?", name).Find(nil)
}

func gormSafeConstantOrder(db *gorm.DB) {
	// SAFE: hardcoded constant, no user input
	db.Order("created_at DESC").Find(nil)
}
