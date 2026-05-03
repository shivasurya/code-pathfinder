// GO-SEC-002 negative test cases — NONE should be detected
package main

import (
	"net/http"
	"os/exec"
)

func safeCommandAllowlist(w http.ResponseWriter, r *http.Request) {
	format := r.FormValue("format")
	// SAFE: allowlist validation — only permit known values
	allowed := map[string]bool{"png": true, "jpg": true, "webp": true}
	if !allowed[format] {
		http.Error(w, "invalid format", 400)
		return
	}
	exec.Command("convert", "input.bmp", "output."+format).Run()
}

func safeCommandHardcoded() {
	// SAFE: no user input in command
	exec.Command("ls", "-la", "/tmp").Run()
}
