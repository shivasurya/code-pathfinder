// GO-REDIRECT-001 negative test cases — NONE should be detected
package main

import "net/http"

func safeRedirectHardcoded(w http.ResponseWriter, r *http.Request) {
	// SAFE: hardcoded constant destination
	http.Redirect(w, r, "/dashboard", http.StatusFound)
}

func safeRedirectValidated(w http.ResponseWriter, r *http.Request) {
	next := r.FormValue("next")
	// SAFE: relative path validation
	if next == "" || next[0] != '/' || (len(next) > 1 && next[1] == '/') {
		next = "/"
	}
	http.Redirect(w, r, next, http.StatusFound)
}
