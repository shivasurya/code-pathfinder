package github

// Comment represents a PR issue comment (not an inline review comment).
//
//nolint:tagliatelle // GitHub REST API uses snake_case JSON field names.
type Comment struct {
	ID   int64  `json:"id"`
	Body string `json:"body"`
	User struct {
		Login string `json:"login"`
	} `json:"user"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	HTMLURL   string `json:"html_url"`
}

// ReviewComment represents an inline review comment on a specific line.
//
//nolint:tagliatelle // GitHub REST API uses snake_case JSON field names.
type ReviewComment struct {
	ID                  int64  `json:"id"`
	PullRequestReviewID int64  `json:"pull_request_review_id"`
	Path                string `json:"path"`
	Line                int    `json:"line"`
	Body                string `json:"body"`
	CreatedAt           string `json:"created_at"`
	UpdatedAt           string `json:"updated_at"`
}

// PullRequest contains PR metadata needed for posting reviews.
type PullRequest struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
	Head   GitRef `json:"head"`
	Base   GitRef `json:"base"`
}

// GitRef is a branch reference with a commit SHA.
type GitRef struct {
	Ref string `json:"ref"`
	SHA string `json:"sha"`
}

// ReviewCommentInput is a single inline comment within a review submission.
type ReviewCommentInput struct {
	Path string `json:"path"`
	Line int    `json:"line"`
	Side string `json:"side"`
	Body string `json:"body"`
}

// createReviewRequest is the POST body for creating a review with inline comments.
//
//nolint:tagliatelle // GitHub REST API uses snake_case JSON field names.
type createReviewRequest struct {
	CommitID string               `json:"commit_id"`
	Body     string               `json:"body"`
	Event    string               `json:"event"`
	Comments []ReviewCommentInput `json:"comments"`
}

// createCommentRequest is the POST body for creating an issue comment.
type createCommentRequest struct {
	Body string `json:"body"`
}

// updateCommentRequest is the PATCH body for updating a comment.
type updateCommentRequest struct {
	Body string `json:"body"`
}

// apiError represents a GitHub API error response.
//
//nolint:tagliatelle // GitHub REST API uses snake_case JSON field names.
type apiError struct {
	Message          string `json:"message"`
	DocumentationURL string `json:"documentation_url"`
}
