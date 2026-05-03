// GO-SSRF-001 negative test cases — NONE should be detected
package main

import "net/http"

func safeSSRFAllowlist(r *http.Request) {
	target := r.FormValue("target")
	// SAFE: validate against allowlist
	allowed := map[string]bool{
		"https://api.example.com": true,
		"https://partner.example.com": true,
	}
	if !allowed[target] {
		return
	}
	http.Get(target)
}

func safeSSRFHardcoded() {
	// SAFE: hardcoded URL, no user input
	http.Get("https://api.example.com/health")
}
