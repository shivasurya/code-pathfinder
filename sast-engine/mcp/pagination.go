package mcp

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// Default and max limits.
const (
	DefaultLimit = 50
	MaxLimit     = 500
)

// PaginationParams holds pagination parameters from request.
type PaginationParams struct {
	Limit  int    `json:"limit"`
	Cursor string `json:"cursor"`
}

// PaginationInfo holds pagination metadata for response.
type PaginationInfo struct {
	Total      int    `json:"total"`
	Returned   int    `json:"returned"`
	HasMore    bool   `json:"hasMore"`
	NextCursor string `json:"nextCursor,omitempty"`
}

// Cursor represents an opaque pagination cursor.
type Cursor struct {
	Offset int    `json:"o"`
	Query  string `json:"q,omitempty"`
}

// EncodeCursor creates an opaque cursor string.
func EncodeCursor(offset int, query string) string {
	c := Cursor{Offset: offset, Query: query}
	bytes, _ := json.Marshal(c)
	return base64.URLEncoding.EncodeToString(bytes)
}

// DecodeCursor parses a cursor string.
func DecodeCursor(cursor string) (*Cursor, error) {
	if cursor == "" {
		return &Cursor{Offset: 0}, nil
	}

	bytes, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor: %w", err)
	}

	var c Cursor
	if err := json.Unmarshal(bytes, &c); err != nil {
		return nil, fmt.Errorf("invalid cursor format: %w", err)
	}

	return &c, nil
}

// ExtractPaginationParams extracts and validates pagination params.
func ExtractPaginationParams(args map[string]any) (*PaginationParams, *RPCError) {
	params := &PaginationParams{
		Limit: DefaultLimit,
	}

	// Extract limit.
	if limitVal, ok := args["limit"]; ok {
		switch v := limitVal.(type) {
		case float64:
			params.Limit = int(v)
		case int:
			params.Limit = v
		default:
			return nil, InvalidParamsError("limit must be a number")
		}
	}

	// Validate limit.
	if params.Limit <= 0 {
		params.Limit = DefaultLimit
	}
	if params.Limit > MaxLimit {
		params.Limit = MaxLimit
	}

	// Extract cursor.
	if cursorVal, ok := args["cursor"].(string); ok {
		params.Cursor = cursorVal
	}

	return params, nil
}

// PaginateSlice applies pagination to a slice of any type.
func PaginateSlice[T any](items []T, params *PaginationParams) ([]T, *PaginationInfo) {
	total := len(items)

	// Decode cursor to get offset.
	cursor, err := DecodeCursor(params.Cursor)
	if err != nil {
		cursor = &Cursor{Offset: 0}
	}

	offset := cursor.Offset
	limit := params.Limit

	// Bounds check.
	if offset >= total {
		return []T{}, &PaginationInfo{
			Total:    total,
			Returned: 0,
			HasMore:  false,
		}
	}

	end := min(offset+limit, total)

	result := items[offset:end]
	hasMore := end < total

	info := &PaginationInfo{
		Total:    total,
		Returned: len(result),
		HasMore:  hasMore,
	}

	if hasMore {
		info.NextCursor = EncodeCursor(end, cursor.Query)
	}

	return result, info
}

// PaginatedResult wraps results with pagination info.
type PaginatedResult struct {
	Items      any            `json:"items"`
	Pagination PaginationInfo `json:"pagination"`
}

// NewPaginatedResult creates a paginated result.
func NewPaginatedResult(items any, info *PaginationInfo) *PaginatedResult {
	return &PaginatedResult{
		Items:      items,
		Pagination: *info,
	}
}
