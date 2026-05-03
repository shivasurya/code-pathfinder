// GO-GORM-SQLI-001 positive test cases — all SHOULD be detected
package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func gormExecInjection(db *gorm.DB, c *gin.Context) {
	id := c.Param("id")                                      // source
	db.Exec("DELETE FROM sessions WHERE user_id = " + id)   // SINK: standalone Exec
}

func gormRawWithStringConcat(db *gorm.DB, r *http.Request) {
	search := r.FormValue("search")                          // source
	query := "SELECT * FROM products WHERE name LIKE '%" + search + "%'"
	db.Raw(query)                                            // SINK: raw query
}

func gormExecViaFormValue(db *gorm.DB, r *http.Request) {
	table := r.FormValue("table")                            // source
	db.Exec("TRUNCATE TABLE " + table)                       // SINK
}

func gormRawViaGinQuery(db *gorm.DB, c *gin.Context) {
	filter := c.Query("where")                               // source
	db.Raw("SELECT * FROM users WHERE " + filter)           // SINK
}
