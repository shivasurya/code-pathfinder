package graph

import (
	"context"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/clike"
	sitter "github.com/smacker/go-tree-sitter"
	clang "github.com/smacker/go-tree-sitter/c"
	cpplang "github.com/smacker/go-tree-sitter/cpp"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/smacker/go-tree-sitter/python"
)

// ProgressCallbacks contains optional callbacks for tracking initialization progress.
type ProgressCallbacks struct {
	// OnStart is called once before processing begins, with the total number of files.
	OnStart func(totalFiles int)
	// OnProgress is called after each file is processed (successfully or with error).
	OnProgress func()
}

// Initialize initializes the code graph by parsing all source files in a directory.
// If callbacks are provided, they will be called to report progress.
func Initialize(directory string, callbacks *ProgressCallbacks) *CodeGraph {
	codeGraph := NewCodeGraph()
	start := time.Now()

	files, err := getFiles(directory)
	if err != nil {
		//nolint:all
		Log("Directory not found:", err)
		return codeGraph
	}

	totalFiles := len(files)

	// Notify start of processing
	if callbacks != nil && callbacks.OnStart != nil {
		callbacks.OnStart(totalFiles)
	}
	numWorkers := 5
	fileChan := make(chan string, totalFiles)
	resultChan := make(chan *CodeGraph, totalFiles)
	var wg sync.WaitGroup

	// Worker function
	worker := func() {
		parser := sitter.NewParser()
		defer parser.Close()

		for file := range fileChan {
			fileName := filepath.Base(file)
			fileExt := filepath.Ext(file)
			fileBase := strings.ToLower(fileName)
			localGraph := NewCodeGraph()

			// Check if it's a Dockerfile or docker-compose file
			isDockerfile := strings.HasPrefix(fileBase, "dockerfile")
			isDockerCompose := strings.Contains(fileBase, "docker-compose") && (fileExt == ".yml" || fileExt == ".yaml")

			if isDockerfile {
				// Handle Dockerfile parsing
				if err := parseDockerfile(file, localGraph); err != nil {
					Log("Error parsing Dockerfile:", err)
					if callbacks != nil && callbacks.OnProgress != nil {
						callbacks.OnProgress()
					}
					continue
				}
				resultChan <- localGraph
				if callbacks != nil && callbacks.OnProgress != nil {
					callbacks.OnProgress()
				}
				continue
			} else if isDockerCompose {
				// Handle docker-compose.yml parsing
				if err := parseDockerCompose(file, localGraph); err != nil {
					Log("Error parsing docker-compose:", err)
					if callbacks != nil && callbacks.OnProgress != nil {
						callbacks.OnProgress()
					}
					continue
				}
				resultChan <- localGraph
				if callbacks != nil && callbacks.OnProgress != nil {
					callbacks.OnProgress()
				}
				continue
			}

			// For .h files, classify as C vs C++ once and cache the result so
			// per-AST-node language checks remain zero-I/O during traversal.
			if fileExt == ".h" {
				clike.CacheHeaderLanguage(file, clike.DetectCppInHeader(file))
			}

			// Handle tree-sitter based parsing for Java, Python, Go, C, and C++.
			// C/C++ cases come first because .h is shared across both grammars and
			// must route via the cached heuristic, not a simple extension match.
			switch {
			case clike.IsCSourceFile(file):
				parser.SetLanguage(clang.GetLanguage())
			case clike.IsCppSourceFile(file):
				parser.SetLanguage(cpplang.GetLanguage())
			case fileExt == ".java":
				parser.SetLanguage(java.GetLanguage())
			case fileExt == ".py":
				parser.SetLanguage(python.GetLanguage())
			case fileExt == ".go":
				parser.SetLanguage(golang.GetLanguage())
			default:
				Log("Unsupported file type:", file)
				if callbacks != nil && callbacks.OnProgress != nil {
					callbacks.OnProgress()
				}
				continue
			}

			sourceCode, err := readFile(file)
			if err != nil {
				Log("File not found:", err)
				if callbacks != nil && callbacks.OnProgress != nil {
					callbacks.OnProgress()
				}
				continue
			}

			tree, err := parser.ParseCtx(context.TODO(), nil, sourceCode)
			if err != nil {
				Log("Error parsing file:", err)
				if callbacks != nil && callbacks.OnProgress != nil {
					callbacks.OnProgress()
				}
				continue
			}
			//nolint:all
			defer tree.Close()

			rootNode := tree.RootNode()
			buildGraphFromAST(rootNode, sourceCode, localGraph, nil, file)

			resultChan <- localGraph
			if callbacks != nil && callbacks.OnProgress != nil {
				callbacks.OnProgress()
			}
		}
		wg.Done()
	}

	// Start workers
	wg.Add(numWorkers)
	for range numWorkers {
		go worker()
	}

	// Send files to workers
	for _, file := range files {
		fileChan <- file
	}
	close(fileChan)

	// Wait for all workers to finish
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results.
	// Each worker already populated edge.From.OutgoingEdges via localGraph.AddEdge,
	// and node pointers are shared across local/global graphs, so we transfer the
	// edge structs without re-attaching them — calling codeGraph.AddEdge here would
	// double every entry in OutgoingEdges and break callers that walk it (e.g. the
	// C/C++ call-graph builders).
	for localGraph := range resultChan {
		for _, node := range localGraph.Nodes {
			codeGraph.AddNode(node)
		}
		codeGraph.Edges = append(codeGraph.Edges, localGraph.Edges...)
	}

	// Resolve transitive inheritance for Python classes.
	// This ensures that classes inheriting from custom enum/interface/dataclass
	// base classes are properly detected as enums/interfaces/dataclasses.
	ResolveTransitiveInheritance(codeGraph)

	end := time.Now()
	elapsed := end.Sub(start)
	Log("Elapsed time: ", elapsed)
	Log("Graph built successfully")

	return codeGraph
}
