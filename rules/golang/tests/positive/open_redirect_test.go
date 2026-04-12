// Open redirect positive test cases — all SHOULD be detected
package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/labstack/echo/v4"
)

// GO-REDIRECT-001: Open redirect via net/http

func openRedirectFormValue(w http.ResponseWriter, r *http.Request) {
	next := r.FormValue("next") // source: user input
	// SINK: attacker controls redirect destination
	http.Redirect(w, r, next, http.StatusFound)
}

func openRedirectURLQuery(w http.ResponseWriter, r *http.Request) {
	to := r.URL.Query().Get("to") // source: URL query param
	http.Redirect(w, r, to, http.StatusMovedPermanently)
}

func openRedirectReferer(w http.ResponseWriter, r *http.Request) {
	referer := r.Referer() // source: Referer header
	// SINK: redirecting to Referer is dangerous — attacker can set Referer header
	http.Redirect(w, r, referer, http.StatusSeeOther)
}

func openRedirectHost(w http.ResponseWriter, r *http.Request) {
	host := r.Host // source: Host header (spoofable)
	http.Redirect(w, r, "https://"+host+"/path", http.StatusTemporaryRedirect)
}

// GO-REDIRECT-001: Open redirect via Gin

func openRedirectGin(c *gin.Context) {
	target := c.Query("target") // source: Gin query param
	// SINK: user controls redirect URL
	c.Redirect(http.StatusFound, target)
}

func openRedirectGinParam(c *gin.Context) {
	next := c.Param("next")    // source: Gin URL param
	c.Redirect(http.StatusFound, next)
}

// GO-REDIRECT-001: Open redirect via Echo

func openRedirectEcho(c echo.Context) error {
	next := c.QueryParam("next") // source: Echo query param
	// SINK: user controls redirect destination
	return c.Redirect(http.StatusFound, next)
}
