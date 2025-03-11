package models

// AnalyzeRequest represents the input for code analysis
type AnalyzeRequest struct {
	JavaSource string `json:"javaSource"`
	Query      string `json:"query"`
}

// QueryResult represents a single result from code analysis
type QueryResult struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Snippet string `json:"snippet"`
}

// AnalyzeResponse represents the response from code analysis
type AnalyzeResponse struct {
	Results []QueryResult `json:"results"`
	Error   string       `json:"error,omitempty"`
}

// Security represents security-related information for a node
type Security struct {
	Risk     string   `json:"risk,omitempty"`
	Impact   string   `json:"impact,omitempty"`
	Category string   `json:"category,omitempty"`
	Rules    []string `json:"rules,omitempty"`
}
