package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestClient creates a Client pointing at a test server.
func newTestClient(t *testing.T, handler http.Handler) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	client := NewClient("test-token", "owner", "repo")
	client.baseURL = srv.URL
	return client
}

func TestNewClient(t *testing.T) {
	c := NewClient("tok", "myowner", "myrepo")
	assert.Equal(t, "tok", c.token)
	assert.Equal(t, "myowner", c.owner)
	assert.Equal(t, "myrepo", c.repo)
	assert.Equal(t, "https://api.github.com", c.baseURL)
	assert.NotNil(t, c.httpClient)
	assert.Equal(t, 30*time.Second, c.httpClient.Timeout)
}

func TestGetPullRequest(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/repos/owner/repo/pulls/42", r.URL.Path)
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
			assert.Equal(t, "application/vnd.github.v3+json", r.Header.Get("Accept"))

			pr := PullRequest{
				Number: 42,
				Title:  "Test PR",
				State:  "open",
				Head:   GitRef{Ref: "feature", SHA: "abc123"},
				Base:   GitRef{Ref: "main", SHA: "def456"},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(pr)
		})

		client := newTestClient(t, handler)
		pr, err := client.GetPullRequest(context.Background(), 42)
		require.NoError(t, err)
		assert.Equal(t, 42, pr.Number)
		assert.Equal(t, "Test PR", pr.Title)
		assert.Equal(t, "abc123", pr.Head.SHA)
		assert.Equal(t, "main", pr.Base.Ref)
	})

	t.Run("not found", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(apiError{Message: "Not Found"})
		})

		client := newTestClient(t, handler)
		_, err := client.GetPullRequest(context.Background(), 999)
		assert.ErrorContains(t, err, "HTTP 404")
		assert.ErrorContains(t, err, "Not Found")
	})

	t.Run("invalid JSON", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("not json"))
		})

		client := newTestClient(t, handler)
		_, err := client.GetPullRequest(context.Background(), 1)
		assert.ErrorContains(t, err, "get pull request")
	})
}

func TestListComments(t *testing.T) {
	t.Run("single page", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/repos/owner/repo/issues/10/comments", r.URL.Path)
			assert.Equal(t, "100", r.URL.Query().Get("per_page"))

			comments := []*Comment{
				{ID: 1, Body: "first"},
				{ID: 2, Body: "second"},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(comments)
		})

		client := newTestClient(t, handler)
		comments, err := client.ListComments(context.Background(), 10)
		require.NoError(t, err)
		require.Len(t, comments, 2)
		assert.Equal(t, int64(1), comments[0].ID)
		assert.Equal(t, "second", comments[1].Body)
	})

	t.Run("pagination", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.Header().Set("Content-Type", "application/json")

			if callCount == 1 {
				assert.Equal(t, "1", r.URL.Query().Get("page"))
				// Return exactly 100 items to trigger next page.
				comments := make([]*Comment, 100)
				for i := range comments {
					comments[i] = &Comment{ID: int64(i), Body: "comment"}
				}
				json.NewEncoder(w).Encode(comments)
			} else {
				assert.Equal(t, "2", r.URL.Query().Get("page"))
				// Return fewer than 100 to stop pagination.
				json.NewEncoder(w).Encode([]*Comment{{ID: 200, Body: "last"}})
			}
		})

		client := newTestClient(t, handler)
		comments, err := client.ListComments(context.Background(), 10)
		require.NoError(t, err)
		assert.Len(t, comments, 101)
		assert.Equal(t, 2, callCount)
	})

	t.Run("empty", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]*Comment{})
		})

		client := newTestClient(t, handler)
		comments, err := client.ListComments(context.Background(), 10)
		require.NoError(t, err)
		assert.Empty(t, comments)
	})

	t.Run("api error", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(apiError{Message: "Bad credentials"})
		})

		client := newTestClient(t, handler)
		_, err := client.ListComments(context.Background(), 10)
		assert.ErrorContains(t, err, "HTTP 401")
		assert.ErrorContains(t, err, "Bad credentials")
	})

	t.Run("invalid JSON", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("not json"))
		})

		client := newTestClient(t, handler)
		_, err := client.ListComments(context.Background(), 1)
		assert.ErrorContains(t, err, "list comments")
	})
}

func TestCreateComment(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/repos/owner/repo/issues/5/comments", r.URL.Path)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			var req createCommentRequest
			require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
			assert.Equal(t, "Hello PR", req.Body)

			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(Comment{ID: 99, Body: req.Body, HTMLURL: "https://github.com/..."})
		})

		client := newTestClient(t, handler)
		comment, err := client.CreateComment(context.Background(), 5, "Hello PR")
		require.NoError(t, err)
		assert.Equal(t, int64(99), comment.ID)
		assert.Equal(t, "Hello PR", comment.Body)
	})

	t.Run("server error", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(apiError{Message: "Internal Server Error"})
		})

		client := newTestClient(t, handler)
		_, err := client.CreateComment(context.Background(), 5, "body")
		assert.ErrorContains(t, err, "HTTP 500")
	})

	t.Run("invalid JSON", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte("{invalid"))
		})

		client := newTestClient(t, handler)
		_, err := client.CreateComment(context.Background(), 1, "body")
		assert.ErrorContains(t, err, "create comment")
	})
}

func TestUpdateComment(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPatch, r.Method)
			assert.Equal(t, "/repos/owner/repo/issues/comments/42", r.URL.Path)

			var req updateCommentRequest
			require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
			assert.Equal(t, "updated body", req.Body)

			json.NewEncoder(w).Encode(Comment{ID: 42, Body: req.Body})
		})

		client := newTestClient(t, handler)
		comment, err := client.UpdateComment(context.Background(), 42, "updated body")
		require.NoError(t, err)
		assert.Equal(t, int64(42), comment.ID)
		assert.Equal(t, "updated body", comment.Body)
	})

	t.Run("not found", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(apiError{Message: "Not Found"})
		})

		client := newTestClient(t, handler)
		_, err := client.UpdateComment(context.Background(), 999, "body")
		assert.ErrorContains(t, err, "HTTP 404")
	})

	t.Run("invalid JSON", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("{bad"))
		})

		client := newTestClient(t, handler)
		_, err := client.UpdateComment(context.Background(), 1, "body")
		assert.ErrorContains(t, err, "update comment")
	})
}

func TestUpdateReviewComment(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPatch, r.Method)
			assert.Equal(t, "/repos/owner/repo/pulls/comments/77", r.URL.Path)

			var req updateCommentRequest
			require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
			assert.Equal(t, "updated inline", req.Body)

			json.NewEncoder(w).Encode(ReviewComment{ID: 77, Body: req.Body, Path: "app.py", Line: 10})
		})

		client := newTestClient(t, handler)
		comment, err := client.UpdateReviewComment(context.Background(), 77, "updated inline")
		require.NoError(t, err)
		assert.Equal(t, int64(77), comment.ID)
		assert.Equal(t, "updated inline", comment.Body)
		assert.Equal(t, "app.py", comment.Path)
	})

	t.Run("not found", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(apiError{Message: "Not Found"})
		})

		client := newTestClient(t, handler)
		_, err := client.UpdateReviewComment(context.Background(), 999, "body")
		assert.ErrorContains(t, err, "HTTP 404")
	})

	t.Run("invalid JSON", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("{bad"))
		})

		client := newTestClient(t, handler)
		_, err := client.UpdateReviewComment(context.Background(), 1, "body")
		assert.ErrorContains(t, err, "update review comment")
	})
}

func TestCreateReview(t *testing.T) {
	t.Run("success with comments", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/repos/owner/repo/pulls/10/reviews", r.URL.Path)

			var req createReviewRequest
			require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
			assert.Equal(t, "abc123", req.CommitID)
			assert.Equal(t, "COMMENT", req.Event)
			assert.Equal(t, "Review body", req.Body)
			require.Len(t, req.Comments, 2)
			assert.Equal(t, "file.py", req.Comments[0].Path)
			assert.Equal(t, 10, req.Comments[0].Line)
			assert.Equal(t, "Issue here", req.Comments[0].Body)

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"id": 1})
		})

		client := newTestClient(t, handler)
		comments := []ReviewCommentInput{
			{Path: "file.py", Line: 10, Body: "Issue here"},
			{Path: "auth.py", Line: 20, Body: "Another issue"},
		}
		err := client.CreateReview(context.Background(), 10, "abc123", "Review body", comments)
		require.NoError(t, err)
	})

	t.Run("validation error", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnprocessableEntity)
			json.NewEncoder(w).Encode(apiError{Message: "Validation Failed"})
		})

		client := newTestClient(t, handler)
		err := client.CreateReview(context.Background(), 10, "sha", "", nil)
		assert.ErrorContains(t, err, "HTTP 422")
		assert.ErrorContains(t, err, "Validation Failed")
	})
}

func TestListReviewComments(t *testing.T) {
	t.Run("single page", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/repos/owner/repo/pulls/7/comments", r.URL.Path)

			comments := []*ReviewComment{
				{ID: 1, Path: "a.py", Line: 5, Body: "issue"},
				{ID: 2, Path: "b.py", Line: 10, Body: "another"},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(comments)
		})

		client := newTestClient(t, handler)
		comments, err := client.ListReviewComments(context.Background(), 7)
		require.NoError(t, err)
		require.Len(t, comments, 2)
		assert.Equal(t, "a.py", comments[0].Path)
		assert.Equal(t, 10, comments[1].Line)
	})

	t.Run("pagination", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			callCount++
			w.Header().Set("Content-Type", "application/json")
			if callCount == 1 {
				comments := make([]*ReviewComment, 100)
				for i := range comments {
					comments[i] = &ReviewComment{ID: int64(i)}
				}
				json.NewEncoder(w).Encode(comments)
			} else {
				json.NewEncoder(w).Encode([]*ReviewComment{{ID: 999}})
			}
		})

		client := newTestClient(t, handler)
		comments, err := client.ListReviewComments(context.Background(), 7)
		require.NoError(t, err)
		assert.Len(t, comments, 101)
		assert.Equal(t, 2, callCount)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("bad"))
		})

		client := newTestClient(t, handler)
		_, err := client.ListReviewComments(context.Background(), 1)
		assert.ErrorContains(t, err, "list review comments")
	})
}

func TestDeleteReviewComment(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodDelete, r.Method)
			assert.Equal(t, "/repos/owner/repo/pulls/comments/55", r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		})

		client := newTestClient(t, handler)
		err := client.DeleteReviewComment(context.Background(), 55)
		require.NoError(t, err)
	})

	t.Run("not found", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(apiError{Message: "Not Found"})
		})

		client := newTestClient(t, handler)
		err := client.DeleteReviewComment(context.Background(), 999)
		assert.ErrorContains(t, err, "HTTP 404")
	})
}

func TestSetBaseURL(t *testing.T) {
	c := NewClient("tok", "o", "r")
	c.SetBaseURL("http://custom.example.com")
	assert.Equal(t, "http://custom.example.com", c.baseURL)
}

func TestGetPullRequest_NetworkError(t *testing.T) {
	client := NewClient("tok", "o", "r")
	client.baseURL = "http://127.0.0.1:1"
	_, err := client.GetPullRequest(context.Background(), 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get pull request")
}

func TestListComments_NetworkError(t *testing.T) {
	client := NewClient("tok", "o", "r")
	client.baseURL = "http://127.0.0.1:1"
	_, err := client.ListComments(context.Background(), 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "list comments")
}

func TestCreateComment_NetworkError(t *testing.T) {
	client := NewClient("tok", "o", "r")
	client.baseURL = "http://127.0.0.1:1"
	_, err := client.CreateComment(context.Background(), 1, "body")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create comment")
}

func TestUpdateComment_NetworkError(t *testing.T) {
	client := NewClient("tok", "o", "r")
	client.baseURL = "http://127.0.0.1:1"
	_, err := client.UpdateComment(context.Background(), 1, "body")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update comment")
}

func TestUpdateReviewComment_NetworkError(t *testing.T) {
	client := NewClient("tok", "o", "r")
	client.baseURL = "http://127.0.0.1:1"
	_, err := client.UpdateReviewComment(context.Background(), 1, "body")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update review comment")
}

func TestCreateReview_NetworkError(t *testing.T) {
	client := NewClient("tok", "o", "r")
	client.baseURL = "http://127.0.0.1:1"
	err := client.CreateReview(context.Background(), 1, "sha", "", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create review")
}

func TestListReviewComments_NetworkError(t *testing.T) {
	client := NewClient("tok", "o", "r")
	client.baseURL = "http://127.0.0.1:1"
	_, err := client.ListReviewComments(context.Background(), 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "list review comments")
}

func TestDeleteReviewComment_NetworkError(t *testing.T) {
	client := NewClient("tok", "o", "r")
	client.baseURL = "http://127.0.0.1:1"
	err := client.DeleteReviewComment(context.Background(), 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "delete review comment")
}

func TestDoRequest_AuthHeaders(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer my-secret", r.Header.Get("Authorization"))
		assert.Equal(t, "application/vnd.github.v3+json", r.Header.Get("Accept"))
		w.WriteHeader(http.StatusOK)
	})

	client := newTestClient(t, handler)
	client.token = "my-secret"

	resp, err := client.doRequest(context.Background(), http.MethodGet, "/test", nil)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDoRequest_ContentTypeOnBody(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var body map[string]string
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "value", body["key"])

		w.WriteHeader(http.StatusOK)
	})

	client := newTestClient(t, handler)

	resp, err := client.doRequest(context.Background(), http.MethodPost, "/test", map[string]string{"key": "value"})
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDoRequest_NoContentTypeWithoutBody(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.Header.Get("Content-Type"))
		w.WriteHeader(http.StatusOK)
	})

	client := newTestClient(t, handler)

	resp, err := client.doRequest(context.Background(), http.MethodGet, "/test", nil)
	require.NoError(t, err)
	defer resp.Body.Close()
}

func TestDoRequest_ContextCancelled(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	})

	client := newTestClient(t, handler)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	resp, err := client.doRequest(ctx, http.MethodGet, "/test", nil) //nolint:bodyclose // Response is nil on error.
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestDoRequest_NetworkError(t *testing.T) {
	client := NewClient("tok", "o", "r")
	client.baseURL = "http://127.0.0.1:1" // Refused port.

	resp, err := client.doRequest(context.Background(), http.MethodGet, "/test", nil) //nolint:bodyclose // Response is nil on error.
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestCheckResponse(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    bool
		errMsg     string
	}{
		{name: "200 OK", statusCode: 200, wantErr: false},
		{name: "201 Created", statusCode: 201, wantErr: false},
		{name: "204 No Content", statusCode: 204, wantErr: false},
		{name: "401 Unauthorized", statusCode: 401, body: `{"message":"Bad credentials"}`, wantErr: true, errMsg: "HTTP 401: Bad credentials"},
		{name: "403 Forbidden", statusCode: 403, body: `{"message":"rate limit"}`, wantErr: true, errMsg: "HTTP 403: rate limit"},
		{name: "500 no JSON", statusCode: 500, body: "not json", wantErr: true, errMsg: "HTTP 500"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			rec.WriteHeader(tt.statusCode)
			if tt.body != "" {
				rec.WriteString(tt.body)
			}
			resp := rec.Result()
			defer resp.Body.Close()

			err := checkResponse(resp)
			if tt.wantErr {
				assert.ErrorContains(t, err, tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
