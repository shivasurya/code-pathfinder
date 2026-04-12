// XSS positive test cases — all of these SHOULD be detected
package main

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
)

// GO-XSS-001: Unsafe template type conversions

func xssTemplateHTML(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name") // source
	safe := template.HTML(name) // SINK: bypasses auto-escaping
	_ = safe
}

func xssTemplateCSS(w http.ResponseWriter, r *http.Request) {
	style := r.FormValue("style")   // source
	safe := template.CSS(style)     // SINK: CSS injection
	_ = safe
}

func xssTemplateJS(w http.ResponseWriter, r *http.Request) {
	code := r.FormValue("code")  // source
	safe := template.JS(code)    // SINK: JS injection
	_ = safe
}

func xssTemplateURL(w http.ResponseWriter, r *http.Request) {
	link := r.FormValue("url")    // source
	safe := template.URL(link)    // SINK: URL injection
	_ = safe
}

func xssTemplateHTMLAttr(w http.ResponseWriter, r *http.Request) {
	attr := r.FormValue("attr")        // source
	safe := template.HTMLAttr(attr)    // SINK: HTML attribute injection
	_ = safe
}

func xssTemplateViaPath(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path              // source: URL path
	safe := template.HTML(path)    // SINK: XSS via path
	_ = safe
}

// GO-XSS-002: fmt.Fprintf to ResponseWriter

func xssFprintfDirect(w http.ResponseWriter, r *http.Request) {
	query := r.FormValue("q")                     // source
	fmt.Fprintf(w, "<p>Results: %s</p>", query)   // SINK: unescaped output
}

func xssFprintlnDirect(w http.ResponseWriter, r *http.Request) {
	msg := r.FormValue("msg")          // source
	fmt.Fprintln(w, "<p>"+msg+"</p>")  // SINK: unescaped output
}

func xssFprintConcatPath(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path                               // source: URL path
	fmt.Fprintf(w, "<div>Path: %s</div>", path)     // SINK: path reflected to HTML
}

func xssFmtSprintfPassedToWrite(w http.ResponseWriter, r *http.Request) {
	user := r.FormValue("user")                            // source
	html := fmt.Sprintf("<h1>Welcome, %s</h1>", user)     // taint propagates
	fmt.Fprint(w, html)                                    // SINK: tainted html written
}

// GO-XSS-003: io.WriteString to ResponseWriter

func xssIOWriteString(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")                 // source
	io.WriteString(w, "<p>Hello, "+name+"</p>") // SINK: unescaped write
}

func xssIOWriteStringViaReferer(w http.ResponseWriter, r *http.Request) {
	referer := r.Referer()                // source: Referer header
	io.WriteString(w, referer)            // SINK: reflected to response
}
