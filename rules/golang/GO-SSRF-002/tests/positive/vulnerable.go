// GO-SSRF-002 positive test cases — all SHOULD be detected
package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func ssrfHTTPGet(c *gin.Context) {
	url := c.Query("url")        // source
	http.Get(url)                // SINK: user-controlled outbound request
}

func ssrfHTTPPost(w http.ResponseWriter, r *http.Request) {
	endpoint := r.FormValue("endpoint") // source
	http.Post(endpoint, "application/json", nil) // SINK
}

func ssrfHTTPClientDo(c *gin.Context) {
	target := c.Query("target")  // source
	client := &http.Client{}
	req, _ := http.NewRequest("GET", target, nil) // SINK
	client.Do(req)
}

func ssrfHTTPGetFormValue(w http.ResponseWriter, r *http.Request) {
	host := r.FormValue("host")
	http.Get("http://" + host + "/api/data") // SINK
}
