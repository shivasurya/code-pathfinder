package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/shivasurya/code-pathfinder/sast-engine/dsl"
	"github.com/shivasurya/code-pathfinder/sast-engine/executor"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/docker"
	"github.com/shivasurya/code-pathfinder/sast-engine/output"
)

// ContainerFile represents a discovered container configuration file.
type ContainerFile struct {
	Path string
	Type string // "dockerfile" or "compose"
}

// DiscoverContainerFiles finds all Dockerfile and docker-compose files in a project.
func DiscoverContainerFiles(projectPath string, logger *output.Logger) ([]ContainerFile, error) {
	var files []ContainerFile

	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err // Propagate error
		}

		if info.IsDir() {
			// Skip common directories
			name := info.Name()
			if name == "node_modules" || name == ".git" || name == "vendor" || name == "__pycache__" {
				return filepath.SkipDir
			}
			return nil
		}

		filename := strings.ToLower(info.Name())

		// Match Dockerfile patterns
		if strings.HasPrefix(filename, "dockerfile") || filename == "dockerfile" {
			files = append(files, ContainerFile{
				Path: path,
				Type: "dockerfile",
			})
			logger.Debug("Found Dockerfile: %s", path)
		}

		// Match docker-compose patterns
		if strings.Contains(filename, "docker-compose") && (strings.HasSuffix(filename, ".yml") || strings.HasSuffix(filename, ".yaml")) {
			files = append(files, ContainerFile{
				Path: path,
				Type: "compose",
			})
			logger.Debug("Found docker-compose: %s", path)
		}

		return nil
	})

	return files, err
}

// CompileContainerRules compiles Python DSL container rules to JSON IR.
func CompileContainerRules(projectRoot string, logger *output.Logger) ([]byte, error) {
	// Find compile script in python-dsl directory
	scriptPath := filepath.Join(projectRoot, "python-dsl", "compile_container_rules.py")

	// Check if script exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("container rules compilation script not found at: %s", scriptPath)
	}

	logger.Debug("Compiling container rules using: %s", scriptPath)

	// Run Python compilation script (suppress warnings) with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "python3", "-W", "ignore", scriptPath)
	cmd.Dir = filepath.Dir(scriptPath)

	// Capture output
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to compile container rules: %w\nOutput: %s", err, string(output))
	}

	logger.Debug("Container rules compiled successfully")

	// Read compiled JSON
	compiledPath := filepath.Join(filepath.Dir(scriptPath), "compiled_rules.json")
	jsonData, err := os.ReadFile(compiledPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read compiled rules: %w", err)
	}

	return jsonData, nil
}

// ScanContainerFiles executes container security rules against discovered files.
func ScanContainerFiles(files []ContainerFile, compiledRules []byte, projectPath string, logger *output.Logger) ([]*dsl.EnrichedDetection, error) {
	if len(files) == 0 {
		return nil, nil
	}

	// Load rules into executor
	exec := &executor.ContainerRuleExecutor{}
	if err := exec.LoadRules(compiledRules); err != nil {
		return nil, fmt.Errorf("failed to load container rules: %w", err)
	}

	var allFindings []*dsl.EnrichedDetection

	// Process Dockerfiles
	dockerfiles := filterByType(files, "dockerfile")
	if len(dockerfiles) > 0 {
		logger.Debug("Scanning %d Dockerfiles", len(dockerfiles))
		for _, file := range dockerfiles {
			findings, err := scanDockerfile(file.Path, exec, projectPath)
			if err != nil {
				logger.Warning("Failed to scan %s: %v", file.Path, err)
				continue
			}
			allFindings = append(allFindings, findings...)
		}
	}

	// Process docker-compose files
	composeFiles := filterByType(files, "compose")
	if len(composeFiles) > 0 {
		logger.Debug("Scanning %d docker-compose files", len(composeFiles))
		for _, file := range composeFiles {
			findings, err := scanComposeFile(file.Path, exec, projectPath)
			if err != nil {
				logger.Warning("Failed to scan %s: %v", file.Path, err)
				continue
			}
			allFindings = append(allFindings, findings...)
		}
	}

	return allFindings, nil
}

func scanDockerfile(filePath string, exec *executor.ContainerRuleExecutor, projectPath string) ([]*dsl.EnrichedDetection, error) {
	parser := docker.NewDockerfileParser()
	dockerfileGraph, err := parser.ParseFile(filePath)
	if err != nil {
		return nil, err
	}

	matches := exec.ExecuteDockerfile(dockerfileGraph)

	// Convert matches to EnrichedDetection format
	findings := make([]*dsl.EnrichedDetection, 0, len(matches))
	for _, match := range matches {
		findings = append(findings, convertToEnrichedDetection(match, filePath, projectPath))
	}

	return findings, nil
}

func scanComposeFile(filePath string, exec *executor.ContainerRuleExecutor, projectPath string) ([]*dsl.EnrichedDetection, error) {
	composeGraph, err := graph.ParseDockerCompose(filePath)
	if err != nil {
		return nil, err
	}

	matches := exec.ExecuteCompose(composeGraph)

	// Convert matches to EnrichedDetection format
	findings := make([]*dsl.EnrichedDetection, 0, len(matches))
	for _, match := range matches {
		findings = append(findings, convertToEnrichedDetection(match, filePath, projectPath))
	}

	return findings, nil
}

func convertToEnrichedDetection(match executor.RuleMatch, filePath string, projectPath string) *dsl.EnrichedDetection {
	// Make path relative to project root
	relPath, err := filepath.Rel(projectPath, filePath)
	if err != nil {
		relPath = filePath
	}

	// Build message with service name if present
	message := match.Message
	if match.ServiceName != "" {
		message = fmt.Sprintf("%s (service: %s)", message, match.ServiceName)
	}

	// Convert CWE string to slice
	var cweList []string
	if match.CWE != "" {
		cweList = []string{match.CWE}
	}

	return &dsl.EnrichedDetection{
		Detection: dsl.DataflowDetection{
			// Container findings don't have dataflow, use basic fields
			FunctionFQN: relPath,
			SinkLine:    match.LineNumber,
			SourceLine:  0,
			Confidence:  1.0,
			Scope:       "container",
		},
		Location: dsl.LocationInfo{
			FilePath: filePath,
			RelPath:  relPath,
			Line:     match.LineNumber,
			Column:   1,
			Function: "",
		},
		Snippet: dsl.CodeSnippet{
			Lines:         []dsl.SnippetLine{},
			StartLine:     match.LineNumber,
			HighlightLine: match.LineNumber,
		},
		Rule: dsl.RuleMetadata{
			ID:          match.RuleID,
			Name:        match.RuleName,
			Severity:    strings.ToLower(match.Severity),
			Description: message,
			CWE:         cweList,
			OWASP:       []string{},
			References:  []string{},
		},
		TaintPath:     nil,                            // Container rules don't have taint paths
		DetectionType: dsl.DetectionTypePattern, // Container rules are pattern-based
	}
}

func filterByType(files []ContainerFile, fileType string) []ContainerFile {
	var filtered []ContainerFile
	for _, f := range files {
		if f.Type == fileType {
			filtered = append(filtered, f)
		}
	}
	return filtered
}

// TryContainerScan attempts to scan container files if they exist and rules are available.
// This is called automatically during normal scanning - no special flags needed.
// Returns nil if container scanning is not available or no container files found.
func TryContainerScan(projectRoot, projectPath string, logger *output.Logger) []*dsl.EnrichedDetection {
	// 1. Discover container files
	containerFiles, err := DiscoverContainerFiles(projectPath, logger)
	if err != nil {
		logger.Debug("Container file discovery failed: %v", err)
		return nil
	}

	if len(containerFiles) == 0 {
		logger.Debug("No container files found in project")
		return nil
	}

	logger.Statistic("Found %d container files (%d Dockerfiles, %d docker-compose)",
		len(containerFiles),
		len(filterByType(containerFiles, "dockerfile")),
		len(filterByType(containerFiles, "compose")))

	// 2. Try to get or compile container rules
	rulesJSON, err := getContainerRulesJSON(projectRoot, logger)
	if err != nil {
		logger.Debug("Container rules not available: %v", err)
		return nil
	}

	// 3. Scan container files
	logger.Progress("Scanning container files...")
	findings, err := ScanContainerFiles(containerFiles, rulesJSON, projectPath, logger)
	if err != nil {
		logger.Warning("Container scan failed: %v", err)
		return nil
	}

	if len(findings) > 0 {
		logger.Statistic("Container scan: %s", getContainerSummary(findings))
	}

	return findings
}

// getContainerRulesJSON loads or compiles container rules.
func getContainerRulesJSON(projectRoot string, logger *output.Logger) ([]byte, error) {
	// Try to find pre-compiled rules first
	compiledPath := filepath.Join(projectRoot, "python-dsl", "compiled_rules.json")

	if _, err := os.Stat(compiledPath); err == nil {
		logger.Debug("Using pre-compiled container rules from: %s", compiledPath)
		return os.ReadFile(compiledPath)
	}

	// If not found, try to compile rules
	logger.Debug("Compiling container rules...")
	return CompileContainerRules(projectRoot, logger)
}

// getContainerSummary builds a summary string for container scan results.
func getContainerSummary(findings []*dsl.EnrichedDetection) string {
	if len(findings) == 0 {
		return "no issues"
	}

	severityCounts := make(map[string]int)
	for _, f := range findings {
		severityCounts[f.Rule.Severity]++
	}

	parts := []string{}
	if count := severityCounts["CRITICAL"]; count > 0 {
		parts = append(parts, fmt.Sprintf("%d critical", count))
	}
	if count := severityCounts["HIGH"]; count > 0 {
		parts = append(parts, fmt.Sprintf("%d high", count))
	}
	if count := severityCounts["MEDIUM"]; count > 0 {
		parts = append(parts, fmt.Sprintf("%d medium", count))
	}
	if count := severityCounts["LOW"]; count > 0 {
		parts = append(parts, fmt.Sprintf("%d low", count))
	}
	if count := severityCounts["INFO"]; count > 0 {
		parts = append(parts, fmt.Sprintf("%d info", count))
	}

	return fmt.Sprintf("%d issues (%s)", len(findings), strings.Join(parts, ", "))
}

// findProjectRoot walks up from projectPath to find the actual project root.
// It looks for python-dsl directory, otherwise uses projectPath.
func findProjectRoot(projectPath string) string {
	current := projectPath
	for {
		pythonDSL := filepath.Join(current, "python-dsl")
		if _, err := os.Stat(pythonDSL); err == nil {
			return current
		}

		parent := filepath.Dir(current)
		if parent == current {
			// Reached filesystem root, use projectPath
			return projectPath
		}
		current = parent
	}
}
