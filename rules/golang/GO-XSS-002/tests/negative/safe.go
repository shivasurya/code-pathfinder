// GO-XSS-002 negative test cases — NONE should be detected
package main

import (
	"fmt"
	"html"
	"html/template"
	"net/http"
)

func safeFprintfStatic(w http.ResponseWriter, r *http.Request) {
	// SAFE: no user input — static strings only
	fmt.Fprintf(w, "<p>Hello, World!</p>")
	fmt.Fprintf(w, "<p>Count: %d</p>", 42)
}

func safeFprintfEscaped(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	// SAFE: escaped before writing
	fmt.Fprintf(w, "<p>Hello, %s</p>", html.EscapeString(name))
}

func safeTemplateExecute(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	tmpl := template.Must(template.New("t").Parse(`<p>Hello, {{.}}</p>`))
	tmpl.Execute(w, name) // SAFE: auto-escaped
}
