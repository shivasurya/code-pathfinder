package diagnostic

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewLLMClient tests Ollama client creation.
func TestNewLLMClient(t *testing.T) {
	client := NewLLMClient("http://localhost:11434", "qwen3-coder:32b")

	assert.NotNil(t, client)
	assert.Equal(t, ProviderOllama, client.Provider)
	assert.Equal(t, "http://localhost:11434", client.BaseURL)
	assert.Equal(t, "qwen3-coder:32b", client.Model)
	assert.Equal(t, 0.0, client.Temperature)
	assert.Equal(t, 2000, client.MaxTokens)
	assert.Equal(t, "", client.APIKey)
	assert.NotNil(t, client.HTTPClient)
}

// TestNewOpenAIClient tests OpenAI-compatible client creation.
func TestNewOpenAIClient(t *testing.T) {
	client := NewOpenAIClient("https://api.x.ai/v1", "grok-beta", "test-api-key")

	assert.NotNil(t, client)
	assert.Equal(t, ProviderOpenAI, client.Provider)
	assert.Equal(t, "https://api.x.ai/v1", client.BaseURL)
	assert.Equal(t, "grok-beta", client.Model)
	assert.Equal(t, "test-api-key", client.APIKey)
	assert.Equal(t, 0.0, client.Temperature)
	assert.Equal(t, 4000, client.MaxTokens)
	assert.NotNil(t, client.HTTPClient)
}

// TestAnalyzeFunction_Success tests successful LLM analysis.
func TestAnalyzeFunction_Success(t *testing.T) {
	// Mock LLM response
	mockResponse := LLMAnalysisResult{
		DiscoveredPatterns: DiscoveredPatterns{
			Sources: []PatternLocation{
				{
					Pattern:     "request.GET",
					Lines:       []int{2},
					Variables:   []string{"user_input"},
					Category:    "user_input",
					Description: "HTTP GET parameter",
				},
			},
			Sinks: []PatternLocation{
				{
					Pattern:     "os.system",
					Lines:       []int{3},
					Variables:   []string{"user_input"},
					Category:    "command_exec",
					Description: "OS command execution",
					Severity:    "CRITICAL",
				},
			},
		},
		DataflowTestCases: []DataflowTestCase{
			{
				TestID:      1,
				Description: "User input to command exec",
				Source: TestCaseSource{
					Pattern:  "request.GET['cmd']",
					Line:     2,
					Variable: "user_input",
				},
				Sink: TestCaseSink{
					Pattern:  "os.system",
					Line:     3,
					Variable: "user_input",
				},
				ExpectedDetection: true,
				VulnerabilityType: "COMMAND_INJECTION",
				Confidence:        0.95,
			},
		},
		AnalysisMetadata: AnalysisMetadata{
			TotalSources: 1,
			TotalSinks:   1,
			TotalFlows:   1,
			Confidence:   0.95,
		},
	}

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		responseBytes, _ := json.Marshal(mockResponse)
		ollamaResp := map[string]any{
			"response": string(responseBytes),
			"done":     true,
		}
		json.NewEncoder(w).Encode(ollamaResp)
	}))
	defer server.Close()

	// Create client pointing to mock server
	client := NewLLMClient(server.URL, "mock-model")

	// Test function
	fn := &FunctionMetadata{
		FQN:        "test.func",
		SourceCode: "def func():\n    pass",
		StartLine:  1,
		EndLine:    2,
	}

	// Analyze
	result, err := client.AnalyzeFunction(fn)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify
	assert.Equal(t, "test.func", result.FunctionFQN)
	assert.Equal(t, 1, len(result.DiscoveredPatterns.Sources))
	assert.Equal(t, 1, len(result.DiscoveredPatterns.Sinks))
	assert.Equal(t, 1, len(result.DataflowTestCases))
	assert.Equal(t, "COMMAND_INJECTION", result.DataflowTestCases[0].VulnerabilityType)
	assert.True(t, result.DataflowTestCases[0].ExpectedDetection)
	assert.Equal(t, "mock-model", result.AnalysisMetadata.ModelUsed)
	assert.NotEmpty(t, result.AnalysisMetadata.ProcessingTime)
}

// TestAnalyzeFunction_InvalidJSON tests error handling for bad JSON.
func TestAnalyzeFunction_InvalidJSON(t *testing.T) {
	// Create mock server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ollamaResp := map[string]any{
			"response": "This is not valid JSON {{{",
			"done":     true,
		}
		json.NewEncoder(w).Encode(ollamaResp)
	}))
	defer server.Close()

	client := NewLLMClient(server.URL, "mock-model")

	fn := &FunctionMetadata{
		FQN:        "test.func",
		SourceCode: "def func():\n    pass",
	}

	result, err := client.AnalyzeFunction(fn)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to parse")
}

// TestAnalyzeFunction_HTTPError tests error handling for HTTP failures.
func TestAnalyzeFunction_HTTPError(t *testing.T) {
	// Create mock server that returns 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal server error"))
	}))
	defer server.Close()

	client := NewLLMClient(server.URL, "mock-model")

	fn := &FunctionMetadata{
		FQN:        "test.func",
		SourceCode: "def func():\n    pass",
	}

	result, err := client.AnalyzeFunction(fn)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "status 500")
}

// TestAnalyzeFunction_MalformedOllamaResponse tests handling of bad Ollama response.
func TestAnalyzeFunction_MalformedOllamaResponse(t *testing.T) {
	// Create mock server that returns malformed Ollama response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not a valid ollama response"))
	}))
	defer server.Close()

	client := NewLLMClient(server.URL, "mock-model")

	fn := &FunctionMetadata{
		FQN:        "test.func",
		SourceCode: "def func():\n    pass",
	}

	result, err := client.AnalyzeFunction(fn)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to parse Ollama response")
}

// TestValidateResult_InvalidConfidence tests confidence validation.
func TestValidateResult_InvalidConfidence(t *testing.T) {
	client := NewLLMClient("http://localhost:11434", "test")

	tests := []struct {
		name       string
		confidence float64
		shouldFail bool
	}{
		{"valid 0.0", 0.0, false},
		{"valid 0.5", 0.5, false},
		{"valid 1.0", 1.0, false},
		{"invalid negative", -0.1, true},
		{"invalid > 1.0", 1.5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &LLMAnalysisResult{
				AnalysisMetadata: AnalysisMetadata{
					Confidence: tt.confidence,
				},
			}

			err := client.validateResult(result)
			if tt.shouldFail {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid confidence")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateResult_InvalidTestCase tests test case validation.
func TestValidateResult_InvalidTestCase(t *testing.T) {
	client := NewLLMClient("http://localhost:11434", "test")

	tests := []struct {
		name       string
		testCase   DataflowTestCase
		shouldFail bool
		errorMsg   string
	}{
		{
			name: "valid test case",
			testCase: DataflowTestCase{
				Source:     TestCaseSource{Line: 5},
				Sink:       TestCaseSink{Line: 10},
				Confidence: 0.9,
			},
			shouldFail: false,
		},
		{
			name: "invalid source line zero",
			testCase: DataflowTestCase{
				Source:     TestCaseSource{Line: 0},
				Sink:       TestCaseSink{Line: 5},
				Confidence: 0.9,
			},
			shouldFail: true,
			errorMsg:   "invalid source line",
		},
		{
			name: "invalid sink line negative",
			testCase: DataflowTestCase{
				Source:     TestCaseSource{Line: 5},
				Sink:       TestCaseSink{Line: -1},
				Confidence: 0.9,
			},
			shouldFail: true,
			errorMsg:   "invalid sink line",
		},
		{
			name: "invalid confidence",
			testCase: DataflowTestCase{
				Source:     TestCaseSource{Line: 5},
				Sink:       TestCaseSink{Line: 10},
				Confidence: 1.5,
			},
			shouldFail: true,
			errorMsg:   "invalid confidence",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &LLMAnalysisResult{
				DataflowTestCases: []DataflowTestCase{tt.testCase},
				AnalysisMetadata: AnalysisMetadata{
					Confidence: 0.9,
				},
			}

			err := client.validateResult(result)
			if tt.shouldFail {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestAnalyzeBatch tests parallel batch processing.
func TestAnalyzeBatch(t *testing.T) {
	// Mock server with counter
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		mockResponse := LLMAnalysisResult{
			AnalysisMetadata: AnalysisMetadata{
				Confidence: 0.9,
			},
		}
		responseBytes, _ := json.Marshal(mockResponse)
		ollamaResp := map[string]any{
			"response": string(responseBytes),
			"done":     true,
		}
		json.NewEncoder(w).Encode(ollamaResp)
	}))
	defer server.Close()

	client := NewLLMClient(server.URL, "mock-model")

	// Create test functions
	functions := []*FunctionMetadata{
		{FQN: "test.func1", SourceCode: "def func1(): pass"},
		{FQN: "test.func2", SourceCode: "def func2(): pass"},
		{FQN: "test.func3", SourceCode: "def func3(): pass"},
	}

	// Analyze batch
	results, errors := client.AnalyzeBatch(functions, 2)

	// Verify
	assert.Equal(t, 3, len(results))
	assert.Equal(t, 0, len(errors))
	assert.Equal(t, 3, callCount)

	assert.NotNil(t, results["test.func1"])
	assert.NotNil(t, results["test.func2"])
	assert.NotNil(t, results["test.func3"])
}

// TestAnalyzeBatch_WithErrors tests batch processing with some failures.
func TestAnalyzeBatch_WithErrors(t *testing.T) {
	// Mock server that fails on certain requests
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read request to determine which function
		var reqBody map[string]any
		json.NewDecoder(r.Body).Decode(&reqBody)
		prompt := reqBody["prompt"].(string)

		// Fail if prompt contains "func2"
		if contains(prompt, "func2") {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Simulated error"))
			return
		}

		mockResponse := LLMAnalysisResult{
			AnalysisMetadata: AnalysisMetadata{
				Confidence: 0.9,
			},
		}
		responseBytes, _ := json.Marshal(mockResponse)
		ollamaResp := map[string]any{
			"response": string(responseBytes),
			"done":     true,
		}
		json.NewEncoder(w).Encode(ollamaResp)
	}))
	defer server.Close()

	client := NewLLMClient(server.URL, "mock-model")

	functions := []*FunctionMetadata{
		{FQN: "test.func1", SourceCode: "def func1(): pass"},
		{FQN: "test.func2", SourceCode: "def func2(): pass"},
		{FQN: "test.func3", SourceCode: "def func3(): pass"},
	}

	results, errors := client.AnalyzeBatch(functions, 2)

	// Should have 2 successes and 1 error
	assert.Equal(t, 2, len(results))
	assert.Equal(t, 1, len(errors))

	assert.NotNil(t, results["test.func1"])
	assert.NotNil(t, results["test.func3"])
	assert.NotNil(t, errors["test.func2"])
}

// Helper function for string contains check.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestAnalyzeFunction_OpenAI tests successful OpenAI API analysis.
func TestAnalyzeFunction_OpenAI(t *testing.T) {
	mockResponse := LLMAnalysisResult{
		DiscoveredPatterns: DiscoveredPatterns{
			Sources: []PatternLocation{
				{Pattern: "input", Lines: []int{1}, Variables: []string{"x"}},
			},
		},
		AnalysisMetadata: AnalysisMetadata{
			Confidence: 0.9,
		},
	}

	// Create mock OpenAI server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify OpenAI request format
		var reqBody map[string]any
		json.NewDecoder(r.Body).Decode(&reqBody)

		assert.Equal(t, "grok-test", reqBody["model"])
		assert.NotNil(t, reqBody["messages"])
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

		// Return OpenAI format response
		responseBytes, _ := json.Marshal(mockResponse)
		openaiResp := map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]string{
						"content": string(responseBytes),
					},
				},
			},
		}
		json.NewEncoder(w).Encode(openaiResp)
	}))
	defer server.Close()

	client := NewOpenAIClient(server.URL, "grok-test", "test-key")

	fn := &FunctionMetadata{
		FQN:        "test.func",
		SourceCode: "def func(): pass",
	}

	result, err := client.AnalyzeFunction(fn)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 1, len(result.DiscoveredPatterns.Sources))
}

// TestAnalyzeFunction_OpenAI_HTTPError tests OpenAI API error handling.
func TestAnalyzeFunction_OpenAI_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "Invalid API key"}`))
	}))
	defer server.Close()

	client := NewOpenAIClient(server.URL, "grok-test", "bad-key")

	fn := &FunctionMetadata{
		FQN:        "test.func",
		SourceCode: "def func(): pass",
	}

	result, err := client.AnalyzeFunction(fn)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "status 401")
}

// TestAnalyzeFunction_OpenAI_NoChoices tests OpenAI response with no choices.
func TestAnalyzeFunction_OpenAI_NoChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		openaiResp := map[string]any{
			"choices": []map[string]any{},
		}
		json.NewEncoder(w).Encode(openaiResp)
	}))
	defer server.Close()

	client := NewOpenAIClient(server.URL, "grok-test", "test-key")

	fn := &FunctionMetadata{
		FQN:        "test.func",
		SourceCode: "def func(): pass",
	}

	result, err := client.AnalyzeFunction(fn)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no choices")
}

// TestExtractJSONFromMarkdown tests JSON extraction from markdown code blocks.
func TestExtractJSONFromMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "json code block",
			input:    "```json\n{\"key\": \"value\"}\n```",
			expected: "\n{\"key\": \"value\"}\n",
		},
		{
			name:     "plain code block",
			input:    "```\n{\"key\": \"value\"}\n```",
			expected: "\n{\"key\": \"value\"}\n",
		},
		{
			name:     "no code block",
			input:    "{\"key\": \"value\"}",
			expected: "{\"key\": \"value\"}",
		},
		{
			name:     "multiple code blocks",
			input:    "```\nfirst\n```\nmiddle\n```\nsecond\n```",
			expected: "\nfirst\n```\nmiddle\n```\nsecond\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSONFromMarkdown(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSaveFailedResponse tests error logging functionality.
func TestSaveFailedResponse(t *testing.T) {
	client := NewLLMClient("http://localhost:11434", "test-model")

	// This test just ensures the function doesn't panic
	// Actual file creation is tested in integration tests
	client.saveFailedResponse("test.func", `{"incomplete": `, assert.AnError)

	// No assertions needed - just verify no panic
}
