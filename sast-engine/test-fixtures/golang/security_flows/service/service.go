package service

import (
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
)

var db *sql.DB

// Convert executes an external command with user-controlled filename — command injection sink.
func Convert(filename string) {
	cmd := exec.Command("convert", filename, "output.png")
	_ = cmd
}

// OpenFile reads a user-controlled path — path traversal sink.
func OpenFile(userPath string) {
	safe := filepath.Join("/srv/files", userPath)
	f, _ := os.Open(safe)
	if f != nil {
		f.Close()
	}
}

// Search executes a SQL query with user-controlled input — SQL injection sink.
func Search(query string) {
	rows, _ := db.Query("SELECT * FROM items WHERE name = '" + query + "'")
	if rows != nil {
		rows.Close()
	}
}
