// GO-SSRF-002 negative test cases — NONE should be detected
package main

import (
	"net/http"
	"net/url"
)

func safeHTTPGetAllowlist(r *http.Request) {
	rawURL := r.FormValue("url")
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return
	}
	allowedHosts := map[string]bool{
		"api.example.com": true,
	}
	if !allowedHosts[parsed.Host] {
		return
	}
	http.Get(rawURL) // safe after allowlist check
}

func safeHTTPGetHardcoded() {
	// SAFE: hardcoded URL, no user input
	http.Get("https://api.example.com/v1/health")
}
