// GO-REDIRECT-001 positive test cases — all SHOULD be detected
package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/labstack/echo/v4"
)

func openRedirectFormValue(w http.ResponseWriter, r *http.Request) {
	next := r.FormValue("next")
	http.Redirect(w, r, next, http.StatusFound) // SINK
}

func openRedirectReferer(w http.ResponseWriter, r *http.Request) {
	referer := r.Referer()
	http.Redirect(w, r, referer, http.StatusSeeOther) // SINK
}

func openRedirectGin(c *gin.Context) {
	target := c.Query("target")
	c.Redirect(http.StatusFound, target) // SINK
}

func openRedirectGinParam(c *gin.Context) {
	next := c.Param("next")
	c.Redirect(http.StatusFound, next) // SINK
}

func openRedirectEcho(c echo.Context) error {
	next := c.QueryParam("next")
	return c.Redirect(http.StatusFound, next) // SINK
}
