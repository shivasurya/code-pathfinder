// GO-XSS-002 positive test cases — all SHOULD be detected
package main

import (
	"fmt"
	"net/http"
)

func xssFprintfDirect(w http.ResponseWriter, r *http.Request) {
	query := r.FormValue("q")                      // source
	fmt.Fprintf(w, "<p>Results: %s</p>", query)    // SINK: unescaped output
}

func xssFprintlnDirect(w http.ResponseWriter, r *http.Request) {
	msg := r.FormValue("msg")           // source
	fmt.Fprintln(w, "<p>"+msg+"</p>")   // SINK
}

func xssFprintConcatPath(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path                                 // source: URL path
	fmt.Fprintf(w, "<div>Path: %s</div>", path)       // SINK
}

func xssFmtSprintfPassedToFprint(w http.ResponseWriter, r *http.Request) {
	user := r.FormValue("user")                             // source
	html := fmt.Sprintf("<h1>Welcome, %s</h1>", user)      // taint propagates
	fmt.Fprint(w, html)                                     // SINK
}
