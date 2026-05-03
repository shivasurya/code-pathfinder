// GO-XSS-003 negative test cases — NONE should be detected
package main

import (
	"html"
	"html/template"
	"io"
	"net/http"
)

func safeIOWriteStringConst(w http.ResponseWriter) {
	// SAFE: constant string, no user input
	io.WriteString(w, "<html><body>Welcome</body></html>")
}

func safeIOWriteStringEscaped(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	// SAFE: escaped before writing
	io.WriteString(w, "<p>"+html.EscapeString(name)+"</p>")
}

func safeTemplateExecuteInstead(w http.ResponseWriter, r *http.Request) {
	msg := r.FormValue("msg")
	// SAFE: template engine handles escaping
	tmpl := template.Must(template.New("t").Parse(`<p>{{.}}</p>`))
	tmpl.Execute(w, msg)
}
