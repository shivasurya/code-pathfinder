// GO-XSS-003 positive test cases — all SHOULD be detected
package main

import (
	"io"
	"net/http"
)

func xssIOWriteString(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")                  // source
	io.WriteString(w, "<p>Hello, "+name+"</p>")  // SINK: unescaped write
}

func xssIOWriteStringReferer(w http.ResponseWriter, r *http.Request) {
	referer := r.Referer()          // source: Referer header
	io.WriteString(w, referer)      // SINK: reflected to response
}

func xssIOWriteStringPath(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path              // source: URL path
	io.WriteString(w, path)         // SINK: path reflected
}
