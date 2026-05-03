package handlers

import (
	"net/http"

	"github.com/example/security_flows/service"
)

// ConvertHandler: r.FormValue("file") → service.Convert(file) → exec.Command
// VULN: GO-SEC-002 command injection
func ConvertHandler(w http.ResponseWriter, r *http.Request) {
	filename := r.FormValue("file")
	service.Convert(filename)
}

// DownloadHandler: r.URL.Path → service.OpenFile(path) → os.Open
// VULN: GO-SEC-003 path traversal
func DownloadHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	service.OpenFile(path)
}

// SearchHandler: r.FormValue("q") → service.Search(q) → sql.DB.Query
// VULN: GO-SEC-001 sql injection
func SearchHandler(w http.ResponseWriter, r *http.Request) {
	query := r.FormValue("q")
	service.Search(query)
}
