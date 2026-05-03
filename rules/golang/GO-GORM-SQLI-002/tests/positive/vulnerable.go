// GO-GORM-SQLI-002 positive test cases — all SHOULD be detected
package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func gormOrderInjection(db *gorm.DB, c *gin.Context) {
	sort := c.Query("sort")  // source
	db.Order(sort)           // SINK: ORDER BY injection
}

func gormWhereInjection(db *gorm.DB, r *http.Request) {
	filter := r.FormValue("filter") // source
	db.Where(filter)                // SINK: WHERE injection
}

func gormGroupInjection(db *gorm.DB, c *gin.Context) {
	group := c.Query("group_by") // source
	db.Group(group)              // SINK: GROUP BY injection
}

func gormHavingInjection(db *gorm.DB, c *gin.Context) {
	having := c.Query("having") // source
	db.Having(having)           // SINK: HAVING injection
}

func gormSelectInjection(db *gorm.DB, r *http.Request) {
	cols := r.FormValue("columns") // source
	db.Select(cols)                // SINK: SELECT column injection
}

func gormJoinsInjection(db *gorm.DB, c *gin.Context) {
	join := c.Query("join") // source
	db.Joins(join)          // SINK: JOIN injection
}
