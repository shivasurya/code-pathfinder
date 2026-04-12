// GO-PATH-001 negative test cases — NONE should be detected
package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func safePathWithValidation(w http.ResponseWriter, r *http.Request) {
	filename := r.FormValue("file")
	// SAFE: clean and validate within base directory
	clean := filepath.Clean(filepath.Join("/var/uploads/", filename))
	if !strings.HasPrefix(clean, "/var/uploads/") {
		http.Error(w, "invalid path", 400)
		return
	}
	content, _ := os.ReadFile(clean)
	w.Write(content)
}

func safePathHardcoded() {
	// SAFE: no user input
	os.ReadFile("/etc/config/app.json")
}
