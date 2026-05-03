// GO-XSS-001 negative test cases — NONE should be detected
package main

import (
	"html"
	"html/template"
	"net/http"
)

func safeTemplateAutoEscape(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	tmpl := template.Must(template.New("t").Parse(`<p>Hello, {{.}}</p>`))
	tmpl.Execute(w, name) // SAFE: auto-escaped by template engine
}

func safeTemplateConstant(w http.ResponseWriter, r *http.Request) {
	// SAFE: hardcoded constant, not user input
	_ = template.HTML("<b>bold</b>")
}

func safeTemplateEscapedFirst(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	escaped := html.EscapeString(name)
	_ = template.HTML("<b>" + escaped + "</b>") // safe: escaped before conversion
}
