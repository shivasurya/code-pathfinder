// GO-PATH-001 positive test cases — all SHOULD be detected
package main

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

func pathTraversalReadFile(w http.ResponseWriter, r *http.Request) {
	filename := r.FormValue("file")                        // source
	content, _ := os.ReadFile("/var/uploads/" + filename)  // SINK: path traversal
	w.Write(content)
}

func pathTraversalOpen(w http.ResponseWriter, r *http.Request) {
	path := r.FormValue("path")          // source
	os.Open(path)                        // SINK
}

func pathTraversalCreate(c *gin.Context) {
	name := c.Param("filename")          // source
	os.Create("/tmp/" + name)            // SINK
}

func pathTraversalJoin(w http.ResponseWriter, r *http.Request) {
	dir := r.FormValue("dir")            // source
	fullPath := filepath.Join("/data", dir) // SINK: join with user input
	os.ReadFile(fullPath)
}
