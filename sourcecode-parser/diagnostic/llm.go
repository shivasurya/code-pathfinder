package diagnostic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// LLMClient handles communication with local LLM (Ollama/vLLM).
type LLMClient struct {
	BaseURL     string
	Model       string
	Temperature float64
	MaxTokens   int
	HTTPClient  *http.Client
}

// NewLLMClient creates a new LLM client.
// Example:
//
//	client := NewLLMClient("http://localhost:11434", "qwen3-coder:32b")
func NewLLMClient(baseURL, model string) *LLMClient {
	return &LLMClient{
		BaseURL:     baseURL,
		Model:       model,
		Temperature: 0.0, // Deterministic
		MaxTokens:   2000,
		HTTPClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// AnalyzeFunction sends a function to the LLM for pattern discovery and test generation.
// Returns structured analysis result or error.
//
// Performance: ~2-5 seconds per function (depends on function size)
//
// Example:
//
//	client := NewLLMClient("http://localhost:11434", "qwen3-coder:32b")
//	result, err := client.AnalyzeFunction(functionMetadata)
//	if err != nil {
//	    log.Printf("LLM analysis failed: %v", err)
//	    return nil, err
//	}
//	fmt.Printf("Found %d sources, %d sinks, %d test cases\n",
//	    len(result.DiscoveredPatterns.Sources),
//	    len(result.DiscoveredPatterns.Sinks),
//	    len(result.DataflowTestCases))
func (c *LLMClient) AnalyzeFunction(fn *FunctionMetadata) (*LLMAnalysisResult, error) {
	startTime := time.Now()

	// Build prompt
	prompt := BuildAnalysisPrompt(fn.SourceCode)

	// Call LLM
	responseText, err := c.callOllama(prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// Parse JSON response
	var result LLMAnalysisResult
	err = json.Unmarshal([]byte(responseText), &result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w\nResponse: %s", err, responseText)
	}

	// Add metadata
	result.FunctionFQN = fn.FQN
	result.AnalysisMetadata.ProcessingTime = time.Since(startTime).String()
	result.AnalysisMetadata.ModelUsed = c.Model

	// Validate result
	if err := c.validateResult(&result); err != nil {
		return nil, fmt.Errorf("invalid LLM result: %w", err)
	}

	return &result, nil
}

// callOllama makes HTTP request to Ollama API.
func (c *LLMClient) callOllama(prompt string) (string, error) {
	// Ollama API format
	requestBody := map[string]interface{}{
		"model":  c.Model,
		"prompt": prompt,
		"stream": false,
		"options": map[string]interface{}{
			"temperature": c.Temperature,
			"num_predict": c.MaxTokens,
		},
		"format": "json", // Request JSON output
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make request
	url := c.BaseURL + "/api/generate"
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("LLM returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Read response
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Parse Ollama response format
	var ollamaResp struct {
		Response string `json:"response"`
		Done     bool   `json:"done"`
	}
	err = json.Unmarshal(bodyBytes, &ollamaResp)
	if err != nil {
		return "", fmt.Errorf("failed to parse Ollama response: %w", err)
	}

	return ollamaResp.Response, nil
}

// validateResult checks that LLM result has required fields.
func (c *LLMClient) validateResult(result *LLMAnalysisResult) error {
	if result.AnalysisMetadata.Confidence < 0.0 || result.AnalysisMetadata.Confidence > 1.0 {
		return fmt.Errorf("invalid confidence: %f", result.AnalysisMetadata.Confidence)
	}

	// Validate test cases
	for i, tc := range result.DataflowTestCases {
		if tc.Source.Line <= 0 {
			return fmt.Errorf("test case %d: invalid source line %d", i, tc.Source.Line)
		}
		if tc.Sink.Line <= 0 {
			return fmt.Errorf("test case %d: invalid sink line %d", i, tc.Sink.Line)
		}
		if tc.Confidence < 0.0 || tc.Confidence > 1.0 {
			return fmt.Errorf("test case %d: invalid confidence %f", i, tc.Confidence)
		}
	}

	return nil
}

// AnalyzeBatch analyzes multiple functions in parallel.
// Returns results map (FQN -> result) and errors map (FQN -> error).
//
// Performance: 4-8 parallel workers, ~30-60 minutes for 10k functions
//
// Example:
//
//	client := NewLLMClient("http://localhost:11434", "qwen3-coder:32b")
//	results, errors := client.AnalyzeBatch(functions, 4)
//	fmt.Printf("Analyzed %d functions, %d errors\n", len(results), len(errors))
func (c *LLMClient) AnalyzeBatch(functions []*FunctionMetadata, concurrency int) (map[string]*LLMAnalysisResult, map[string]error) {
	results := make(map[string]*LLMAnalysisResult)
	errors := make(map[string]error)

	// Channel for work
	workChan := make(chan *FunctionMetadata, len(functions))
	resultChan := make(chan struct {
		fqn    string
		result *LLMAnalysisResult
		err    error
	}, len(functions))

	// Start workers
	for i := 0; i < concurrency; i++ {
		go func() {
			for fn := range workChan {
				result, err := c.AnalyzeFunction(fn)
				resultChan <- struct {
					fqn    string
					result *LLMAnalysisResult
					err    error
				}{fn.FQN, result, err}
			}
		}()
	}

	// Send work
	for _, fn := range functions {
		workChan <- fn
	}
	close(workChan)

	// Collect results
	for i := 0; i < len(functions); i++ {
		res := <-resultChan
		if res.err != nil {
			errors[res.fqn] = res.err
		} else {
			results[res.fqn] = res.result
		}
	}

	return results, errors
}
