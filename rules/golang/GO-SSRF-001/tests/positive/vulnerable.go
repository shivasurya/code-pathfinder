// GO-SSRF-001 positive test cases — all SHOULD be detected
package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func ssrfViaGet(c *gin.Context) {
	target := c.Query("url")             // source
	http.Get(target)                     // SINK: user controls outbound URL
}

func ssrfViaFormValue(w http.ResponseWriter, r *http.Request) {
	target := r.FormValue("target")      // source
	http.Get(target)                     // SINK: SSRF
}

func ssrfViaPost(c *gin.Context) {
	endpoint := c.Query("endpoint")      // source
	http.Post(endpoint, "application/json", nil) // SINK
}

func ssrfViaQueryParam(w http.ResponseWriter, r *http.Request) {
	host := r.FormValue("host")
	client := &http.Client{}
	req, _ := http.NewRequest("GET", "https://"+host+"/api", nil) // SINK
	client.Do(req)
}
