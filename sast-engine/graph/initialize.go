package graph

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/smacker/go-tree-sitter/python"
)

// Initialize initializes the code graph by parsing all source files in a directory.
func Initialize(directory string) *CodeGraph {
	codeGraph := NewCodeGraph()
	start := time.Now()

	files, err := getFiles(directory)
	if err != nil {
		//nolint:all
		Log("Directory not found:", err)
		return codeGraph
	}

	totalFiles := len(files)
	numWorkers := 5
	fileChan := make(chan string, totalFiles)
	resultChan := make(chan *CodeGraph, totalFiles)
	statusChan := make(chan string, numWorkers)
	progressChan := make(chan int, totalFiles)
	var wg sync.WaitGroup

	// Worker function
	worker := func(workerID int) {
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
				statusChan <- fmt.Sprintf("\033[32mWorker %d ....... Parsing Dockerfile %s\033[0m", workerID, fileName)
				if err := parseDockerfile(file, localGraph); err != nil {
					Log("Error parsing Dockerfile:", err)
					continue
				}
				statusChan <- fmt.Sprintf("\033[32mWorker %d ....... Done processing Dockerfile %s\033[0m", workerID, fileName)
				resultChan <- localGraph
				progressChan <- 1
				continue
			} else if isDockerCompose {
				// Handle docker-compose.yml parsing
				statusChan <- fmt.Sprintf("\033[32mWorker %d ....... Parsing docker-compose %s\033[0m", workerID, fileName)
				if err := parseDockerCompose(file, localGraph); err != nil {
					Log("Error parsing docker-compose:", err)
					continue
				}
				statusChan <- fmt.Sprintf("\033[32mWorker %d ....... Done processing docker-compose %s\033[0m", workerID, fileName)
				resultChan <- localGraph
				progressChan <- 1
				continue
			}

			// Handle tree-sitter based parsing for Java and Python
			switch fileExt {
			case ".java":
				parser.SetLanguage(java.GetLanguage())
			case ".py":
				parser.SetLanguage(python.GetLanguage())
			default:
				Log("Unsupported file type:", file)
				continue
			}

			statusChan <- fmt.Sprintf("\033[32mWorker %d ....... Reading and parsing code %s\033[0m", workerID, fileName)
			sourceCode, err := readFile(file)
			if err != nil {
				Log("File not found:", err)
				continue
			}

			tree, err := parser.ParseCtx(context.TODO(), nil, sourceCode)
			if err != nil {
				Log("Error parsing file:", err)
				continue
			}
			//nolint:all
			defer tree.Close()

			rootNode := tree.RootNode()
			statusChan <- fmt.Sprintf("\033[32mWorker %d ....... Building graph and traversing code %s\033[0m", workerID, fileName)
			buildGraphFromAST(rootNode, sourceCode, localGraph, nil, file)
			statusChan <- fmt.Sprintf("\033[32mWorker %d ....... Done processing file %s\033[0m", workerID, fileName)

			resultChan <- localGraph
			progressChan <- 1
		}
		wg.Done()
	}

	// Start workers
	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go worker(i + 1)
	}

	// Send files to workers
	for _, file := range files {
		fileChan <- file
	}
	close(fileChan)

	// Status updater
	go func() {
		statusLines := make([]string, numWorkers)
		progress := 0
		for {
			select {
			case status, ok := <-statusChan:
				if !ok {
					return
				}
				workerID := int(status[12] - '0')
				statusLines[workerID-1] = status
			case _, ok := <-progressChan:
				if !ok {
					return
				}
				progress++
			}
			// Only print progress in verbose mode to avoid polluting structured output
			if verboseFlag {
				fmt.Print("\033[H\033[J") // Clear the screen
			}
			for _, line := range statusLines {
				Log(line)
			}
			Fmt("Progress: %d%%\n", (progress*100)/totalFiles)
		}
	}()

	// Wait for all workers to finish
	go func() {
		wg.Wait()
		close(resultChan)
		close(statusChan)
		close(progressChan)
	}()

	// Collect results
	for localGraph := range resultChan {
		for _, node := range localGraph.Nodes {
			codeGraph.AddNode(node)
		}
		for _, edge := range localGraph.Edges {
			codeGraph.AddEdge(edge.From, edge.To)
		}
	}

	end := time.Now()
	elapsed := end.Sub(start)
	Log("Elapsed time: ", elapsed)
	Log("Graph built successfully")

	return codeGraph
}
