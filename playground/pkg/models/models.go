package models

// AnalyzeRequest represents the request body for code analysis
type AnalyzeRequest struct {
	JavaSource string `json:"javaSource"`
	Query      string `json:"query"`
	Category   string `json:"category"` // Technology or language-based category
}

// AnalyzeResponse represents the response from code analysis
type AnalyzeResponse struct {
	Results []QueryResult `json:"results"`
}

// QueryResult represents a single result from query execution
type QueryResult struct {
	File    string `json:"file"`
	Line    int64  `json:"line"`
	Snippet string `json:"snippet"`
	Kind    string `json:"kind,omitempty"` // Type of the node or result
}
