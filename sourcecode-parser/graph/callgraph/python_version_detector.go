package callgraph

import (
	cgbuilder "github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/builder"
)

// detectPythonVersion infers Python version from project files.
//
// Deprecated: This function now delegates to builder.DetectPythonVersion.
func detectPythonVersion(projectPath string) string {
	return cgbuilder.DetectPythonVersion(projectPath)
}
