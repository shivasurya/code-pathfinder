// XSS negative test cases — NONE of these should be detected
package main

import (
	"fmt"
	"html"
	"html/template"
	"io"
	"net/http"
)

// SAFE: constant strings passed to template type conversions
func safeTemplateHTMLConstant() {
	// OK: hardcoded constant — not user input
	_ = template.HTML("<b>bold</b>")
}

func safeTemplateHTMLEscaped(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	escaped := html.EscapeString(name)   // sanitizer: html.EscapeString
	_ = template.HTML("<b>" + escaped + "</b>")  // safe: escaped before conversion
}

// SAFE: template auto-escape (using Execute, not type conversion)
func safeTemplateExecute(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.New("t").Parse("<p>Hello, {{.}}</p>"))
	name := r.FormValue("name")
	tmpl.Execute(w, name) // SAFE: html/template auto-escapes on Execute
}

// SAFE: fmt.Fprintf with no user input (static strings only)
func safeFprintfStatic(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<p>Hello, World!</p>")          // SAFE: no user input
	fmt.Fprintf(w, "<p>Count: %d</p>", 42)           // SAFE: integer literal
}

// SAFE: fmt.Fprintf writing to a file, not ResponseWriter
func safeFprintfToFile() {
	// Not writing to ResponseWriter — not an XSS risk in this context
	// (though could be other issues)
	var buf fmt.Stringer
	_ = buf
}

// SAFE: io.WriteString with constant
func safeIOWriteStringConst(w http.ResponseWriter) {
	io.WriteString(w, "<html><body>static</body></html>") // SAFE: constant
}

// SAFE: user data written through template engine (auto-escapes)
func safeViaTemplateEngine(w http.ResponseWriter, r *http.Request) {
	tmpl, _ := template.ParseFiles("index.html")
	data := struct{ Name string }{Name: r.FormValue("name")}
	tmpl.Execute(w, data) // SAFE: template engine escapes data
}

// SAFE: JSON response via encoding/json (user data properly encoded)
func safeJSONResponse(w http.ResponseWriter, r *http.Request) {
	// SAFE: encoding/json.Encode handles escaping — not using fmt.Fprintf with user data
	name := r.FormValue("name")
	w.Header().Set("Content-Type", "application/json")
	// No fmt.Fprintf with user data — using json encoder
	type resp struct{ Name string }
	_ = name // used via json.Encoder below in real code
}
