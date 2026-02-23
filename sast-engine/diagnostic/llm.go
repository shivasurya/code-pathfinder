package diagnostic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// LLMProvider represents the type of LLM provider.
type LLMProvider string

const (
	ProviderOllama LLMProvider = "ollama"
	ProviderOpenAI LLMProvider = "openai" // Also compatible with xAI Grok, vLLM, etc.
)

// LLMClient handles communication with LLM providers (Ollama, OpenAI-compatible APIs).
type LLMClient struct {
	Provider    LLMProvider
	BaseURL     string
	Model       string
	Temperature float64
	MaxTokens   int
	APIKey      string // For OpenAI-compatible APIs (xAI Grok, etc.)
	HTTPClient  *http.Client
}

// NewLLMClient creates a new LLM client for Ollama.
// Example:
//
//	client := NewLLMClient("http://localhost:11434", "qwen3-coder:32b")
func NewLLMClient(baseURL, model string) *LLMClient {
	return &LLMClient{
		Provider:    ProviderOllama,
		BaseURL:     baseURL,
		Model:       model,
		Temperature: 0.0, // Deterministic
		MaxTokens:   2000,
		HTTPClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// NewOpenAIClient creates a new OpenAI-compatible client (xAI Grok, vLLM, etc.).
// Example:
//
//	client := NewOpenAIClient("https://api.x.ai/v1", "grok-beta", "xai-YOUR_API_KEY")
func NewOpenAIClient(baseURL, model, apiKey string) *LLMClient {
	return &LLMClient{
		Provider:    ProviderOpenAI,
		BaseURL:     baseURL,
		Model:       model,
		APIKey:      apiKey,
		Temperature: 0.0,  // Deterministic
		MaxTokens:   4000, // Increased for complex functions
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

	// Call LLM based on provider
	var responseText string
	var err error
	switch c.Provider {
	case ProviderOllama:
		responseText, err = c.callOllama(prompt)
	case ProviderOpenAI:
		responseText, err = c.callOpenAI(prompt)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", c.Provider)
	}

	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// Parse JSON response
	var result LLMAnalysisResult
	err = json.Unmarshal([]byte(responseText), &result)
	if err != nil {
		// Try to extract JSON from markdown code blocks if present
		responseText = extractJSONFromMarkdown(responseText)
		err = json.Unmarshal([]byte(responseText), &result)
		if err != nil {
			// Save failed response to debug file
			c.saveFailedResponse(fn.FQN, responseText, err)

			// Log first 500 chars for debugging
			preview := responseText
			if len(preview) > 500 {
				preview = preview[:500]
			}
			return nil, fmt.Errorf("failed to parse LLM response: %w\nResponse preview: %s", err, preview)
		}
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
	requestBody := map[string]any{
		"model":  c.Model,
		"prompt": prompt,
		"stream": false,
		"options": map[string]any{
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

// callOpenAI makes HTTP request to OpenAI-compatible API (xAI Grok, vLLM, etc.).
func (c *LLMClient) callOpenAI(prompt string) (string, error) {
	// OpenAI API format
	requestBody := map[string]any{
		"model": c.Model,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"temperature":     c.Temperature,
		"max_tokens":      c.MaxTokens,
		"response_format": map[string]string{"type": "json_object"}, // Request JSON output
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make request
	url := c.BaseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

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

	// Parse OpenAI response format
	var openaiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	err = json.Unmarshal(bodyBytes, &openaiResp)
	if err != nil {
		return "", fmt.Errorf("failed to parse OpenAI response: %w", err)
	}

	if len(openaiResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in OpenAI response")
	}

	return openaiResp.Choices[0].Message.Content, nil
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
	for range concurrency {
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
	for range functions {
		res := <-resultChan
		if res.err != nil {
			errors[res.fqn] = res.err
		} else {
			results[res.fqn] = res.result
		}
	}

	return results, errors
}

// extractJSONFromMarkdown extracts JSON from markdown code blocks.
func extractJSONFromMarkdown(text string) string {
	// Try to find JSON between ```json and ```
	start := -1
	end := -1

	// Look for ```json
	jsonMarker := "```json"
	idx := len(jsonMarker)
	if len(text) > idx && text[:idx] == jsonMarker {
		start = idx
	}

	// Look for closing ```
	if start != -1 {
		closingMarker := "```"
		closeIdx := len(text) - len(closingMarker)
		if closeIdx > start && text[closeIdx:] == closingMarker {
			end = closeIdx
		}
	}

	if start != -1 && end != -1 {
		return text[start:end]
	}

	// Try plain ``` markers
	markers := []int{}
	for i := 0; i < len(text)-2; i++ {
		if text[i:i+3] == "```" {
			markers = append(markers, i)
		}
	}

	if len(markers) >= 2 {
		// Return content between first and last markers
		return text[markers[0]+3 : markers[len(markers)-1]]
	}

	return text
}

// saveFailedResponse saves failed LLM response to debug file.
func (c *LLMClient) saveFailedResponse(fqn, responseText string, parseErr error) {
	// Create debug directory
	debugDir := "/tmp/diagnostic_llm_errors"
	os.MkdirAll(debugDir, 0755)

	// Create timestamped filename
	timestamp := time.Now().Format("20060102_150405")
	filename := filepath.Join(debugDir, fmt.Sprintf("error_%s.txt", timestamp))

	// Write error details
	content := fmt.Sprintf("=== LLM Response Parse Error ===\n")
	content += fmt.Sprintf("Function: %s\n", fqn)
	content += fmt.Sprintf("Error: %v\n", parseErr)
	content += fmt.Sprintf("Provider: %s\n", c.Provider)
	content += fmt.Sprintf("Model: %s\n", c.Model)
	content += fmt.Sprintf("\n=== Full Response ===\n%s\n", responseText)

	os.WriteFile(filename, []byte(content), 0644)
}
