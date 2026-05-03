// GO-SEC-002 positive test cases — all SHOULD be detected
package main

import (
	"net/http"
	"os/exec"

	"github.com/gin-gonic/gin"
)

func commandInjectionFilename(w http.ResponseWriter, r *http.Request) {
	filename := r.FormValue("file")              // source
	cmd := exec.Command("convert", filename)     // SINK: user controls argument
	cmd.Run()
}

func commandInjectionViaGin(c *gin.Context) {
	model := c.Query("model")                    // source
	exec.Command("ollama", "pull", model).Run()  // SINK
}

func commandInjectionBashC(w http.ResponseWriter, r *http.Request) {
	script := r.FormValue("cmd")                 // source
	// SINK: user controls bash -c argument — full shell injection
	exec.Command("bash", "-c", script).Run()
}

func commandInjectionContextFlag(w http.ResponseWriter, r *http.Request) {
	flag := r.FormValue("flag")
	exec.Command("tool", "--option="+flag).Run() // SINK
}
