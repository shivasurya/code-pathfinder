package extraction

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/resolution"
)

// FunctionJob represents a function to process for return type extraction.
type FunctionJob struct {
	FQN        string
	ReturnType string
	File       string
}

// ExtractGoReturnTypes extracts return type information from indexed functions.
//
// This function implements Pass 2a of the Go call graph construction pipeline.
// It processes functions that were indexed in Pass 1, parsing their return type
// strings (stored in node.ReturnType) into structured TypeInfo objects.
//
// PERFORMANCE: Uses parallel worker pool to process functions concurrently.
// Progress is reported to stderr every 500 functions processed.
//
// Algorithm:
//  1. Collect all functions with return types into a job queue
//  2. Spawn worker goroutines (based on CPU count)
//  3. Each worker:
//     a) Reads jobs from queue
//     b) Parses return type using ParseGoTypeString()
//     c) Stores result in typeEngine (thread-safe with mutex)
//  4. Report progress every 500 functions
//
// Thread Safety:
//
//	This function is thread-safe because typeEngine.AddReturnType() uses mutexes.
//	Multiple workers can call it concurrently.
//
// Parameters:
//   - callGraph: The call graph with indexed functions (from Pass 1)
//   - registry: Go module registry for type resolution (from Phase 1)
//   - typeEngine: Type inference engine to store results (from PR-13)
//   - showProgress: If true, prints progress to stderr
//
// Returns:
//   - error: Currently always returns nil (errors are logged but not propagated)
func ExtractGoReturnTypes(
	callGraph *core.CallGraph,
	registry *core.GoModuleRegistry,
	typeEngine *resolution.GoTypeInferenceEngine,
) error {
	return ExtractGoReturnTypesWithProgress(callGraph, registry, typeEngine, true)
}

// ExtractGoReturnTypesWithProgress is the internal implementation with progress control.
func ExtractGoReturnTypesWithProgress(
	callGraph *core.CallGraph,
	registry *core.GoModuleRegistry,
	typeEngine *resolution.GoTypeInferenceEngine,
	showProgress bool,
) error {
	// Collect all functions that have return types
	jobs := make([]*FunctionJob, 0, len(callGraph.Functions))
	for fqn, node := range callGraph.Functions {
		if node.ReturnType != "" {
			jobs = append(jobs, &FunctionJob{
				FQN:        fqn,
				ReturnType: node.ReturnType,
				File:       node.File,
			})
		}
	}

	totalJobs := len(jobs)
	if totalJobs == 0 {
		return nil
	}

	// Determine worker count (same as Python builder)
	numWorkers := max(runtime.NumCPU()*3/4, 1)
	if numWorkers > 8 {
		numWorkers = 8 // Cap at 8 for diminishing returns
	}

	// Create job channel
	jobChan := make(chan *FunctionJob, 100)
	var processed atomic.Int64
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < numWorkers; i++ {
		wg.Go(func() {
			for job := range jobChan {
				// Parse return type
				typeInfo, err := ParseGoTypeString(job.ReturnType, registry, job.File)
				if err == nil && typeInfo != nil {
					typeEngine.AddReturnType(job.FQN, typeInfo)
				}

				// Progress tracking
				count := processed.Add(1)
				if showProgress && count%500 == 0 {
					percentage := float64(count) / float64(totalJobs) * 100
					fmt.Fprintf(os.Stderr, "\r    Return types: %d/%d (%.1f%%)", count, totalJobs, percentage)
				}
			}
		})
	}

	// Queue all jobs
	for _, job := range jobs {
		jobChan <- job
	}
	close(jobChan)
	wg.Wait()

	// Final progress
	if showProgress && totalJobs > 0 {
		fmt.Fprintf(os.Stderr, "\r    Return types: %d/%d (100.0%%)\n", totalJobs, totalJobs)
	}

	return nil
}
