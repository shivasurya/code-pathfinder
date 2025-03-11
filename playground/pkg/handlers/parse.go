package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/shivasurya/code-pathfinder/playground/pkg/ast"
	"github.com/shivasurya/code-pathfinder/playground/pkg/models"
	"github.com/shivasurya/code-pathfinder/playground/pkg/utils"
)

// ParseHandler processes POST requests to /parse endpoint.
// It accepts Java source code, parses it into an AST using tree-sitter,
// and returns the AST structure for visualization.
func ParseHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer utils.LogRequestDuration("parseHandler", start)

	if !utils.ValidateMethod(w, r, http.MethodPost) {
		return
	}

	var req models.ParseRequest
	if err := utils.DecodeJSONRequest(w, r, &req); err != nil {
		return
	}

	// Validate source code is not empty
	if strings.TrimSpace(req.JavaSource) == "" {
		utils.SendErrorResponse(w, "Source code cannot be empty", nil)
		return
	}

	// Parse the Java source code
	ast, err := ast.ParseJavaSource(req.JavaSource)
	if err != nil {
		utils.SendErrorResponse(w, "Failed to parse source code", err)
		return
	}

	utils.SendJSONResponse(w, models.ParseResponse{AST: ast})
}
