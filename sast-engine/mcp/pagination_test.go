package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeCursor(t *testing.T) {
	cursor := EncodeCursor(100, "test")

	assert.NotEmpty(t, cursor)
	// Should be URL-safe base64.
	assert.NotContains(t, cursor, "+")
	assert.NotContains(t, cursor, "/")
}

func TestDecodeCursor(t *testing.T) {
	// Encode then decode.
	encoded := EncodeCursor(50, "query")
	decoded, err := DecodeCursor(encoded)

	require.NoError(t, err)
	assert.Equal(t, 50, decoded.Offset)
	assert.Equal(t, "query", decoded.Query)
}

func TestDecodeCursor_Empty(t *testing.T) {
	decoded, err := DecodeCursor("")

	require.NoError(t, err)
	assert.Equal(t, 0, decoded.Offset)
}

func TestDecodeCursor_Invalid(t *testing.T) {
	_, err := DecodeCursor("not-valid-base64!!!")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid cursor")
}

func TestDecodeCursor_InvalidJSON(t *testing.T) {
	// Valid base64 but invalid JSON.
	_, err := DecodeCursor("bm90anNvbg==") // "notjson"
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid cursor format")
}

func TestExtractPaginationParams_Defaults(t *testing.T) {
	args := map[string]any{}
	params, err := ExtractPaginationParams(args)

	assert.Nil(t, err)
	assert.Equal(t, DefaultLimit, params.Limit)
	assert.Empty(t, params.Cursor)
}

func TestExtractPaginationParams_WithLimit(t *testing.T) {
	args := map[string]any{
		"limit": float64(25),
	}
	params, err := ExtractPaginationParams(args)

	assert.Nil(t, err)
	assert.Equal(t, 25, params.Limit)
}

func TestExtractPaginationParams_LimitInt(t *testing.T) {
	args := map[string]any{
		"limit": 30,
	}
	params, err := ExtractPaginationParams(args)

	assert.Nil(t, err)
	assert.Equal(t, 30, params.Limit)
}

func TestExtractPaginationParams_LimitCapped(t *testing.T) {
	args := map[string]any{
		"limit": float64(10000),
	}
	params, err := ExtractPaginationParams(args)

	assert.Nil(t, err)
	assert.Equal(t, MaxLimit, params.Limit)
}

func TestExtractPaginationParams_InvalidLimit(t *testing.T) {
	args := map[string]any{
		"limit": "not a number",
	}
	_, err := ExtractPaginationParams(args)

	assert.NotNil(t, err)
	assert.Equal(t, ErrCodeInvalidParams, err.Code)
}

func TestExtractPaginationParams_NegativeLimit(t *testing.T) {
	args := map[string]any{
		"limit": float64(-5),
	}
	params, err := ExtractPaginationParams(args)

	assert.Nil(t, err)
	assert.Equal(t, DefaultLimit, params.Limit) // Reset to default.
}

func TestExtractPaginationParams_ZeroLimit(t *testing.T) {
	args := map[string]any{
		"limit": float64(0),
	}
	params, err := ExtractPaginationParams(args)

	assert.Nil(t, err)
	assert.Equal(t, DefaultLimit, params.Limit) // Reset to default.
}

func TestExtractPaginationParams_WithCursor(t *testing.T) {
	cursor := EncodeCursor(100, "")
	args := map[string]any{
		"cursor": cursor,
	}
	params, err := ExtractPaginationParams(args)

	assert.Nil(t, err)
	assert.Equal(t, cursor, params.Cursor)
}

func TestPaginateSlice_FirstPage(t *testing.T) {
	items := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}
	params := &PaginationParams{Limit: 3, Cursor: ""}

	result, info := PaginateSlice(items, params)

	assert.Equal(t, []string{"a", "b", "c"}, result)
	assert.Equal(t, 10, info.Total)
	assert.Equal(t, 3, info.Returned)
	assert.True(t, info.HasMore)
	assert.NotEmpty(t, info.NextCursor)
}

func TestPaginateSlice_MiddlePage(t *testing.T) {
	items := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}
	cursor := EncodeCursor(3, "")
	params := &PaginationParams{Limit: 3, Cursor: cursor}

	result, info := PaginateSlice(items, params)

	assert.Equal(t, []string{"d", "e", "f"}, result)
	assert.Equal(t, 10, info.Total)
	assert.Equal(t, 3, info.Returned)
	assert.True(t, info.HasMore)
}

func TestPaginateSlice_LastPage(t *testing.T) {
	items := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}
	cursor := EncodeCursor(9, "")
	params := &PaginationParams{Limit: 3, Cursor: cursor}

	result, info := PaginateSlice(items, params)

	assert.Equal(t, []string{"j"}, result)
	assert.Equal(t, 10, info.Total)
	assert.Equal(t, 1, info.Returned)
	assert.False(t, info.HasMore)
	assert.Empty(t, info.NextCursor)
}

func TestPaginateSlice_PastEnd(t *testing.T) {
	items := []string{"a", "b", "c"}
	cursor := EncodeCursor(100, "")
	params := &PaginationParams{Limit: 10, Cursor: cursor}

	result, info := PaginateSlice(items, params)

	assert.Empty(t, result)
	assert.Equal(t, 3, info.Total)
	assert.Equal(t, 0, info.Returned)
	assert.False(t, info.HasMore)
}

func TestPaginateSlice_ExactFit(t *testing.T) {
	items := []string{"a", "b", "c"}
	params := &PaginationParams{Limit: 3, Cursor: ""}

	result, info := PaginateSlice(items, params)

	assert.Equal(t, []string{"a", "b", "c"}, result)
	assert.False(t, info.HasMore)
}

func TestPaginateSlice_EmptySlice(t *testing.T) {
	items := []string{}
	params := &PaginationParams{Limit: 10, Cursor: ""}

	result, info := PaginateSlice(items, params)

	assert.Empty(t, result)
	assert.Equal(t, 0, info.Total)
	assert.False(t, info.HasMore)
}

func TestPaginateSlice_InvalidCursor(t *testing.T) {
	items := []string{"a", "b", "c"}
	params := &PaginationParams{Limit: 2, Cursor: "invalid!!!"}

	result, info := PaginateSlice(items, params)

	// Should start from beginning on invalid cursor.
	assert.Equal(t, []string{"a", "b"}, result)
	assert.Equal(t, 3, info.Total)
}

func TestNewPaginatedResult(t *testing.T) {
	items := []string{"a", "b"}
	info := &PaginationInfo{Total: 10, Returned: 2, HasMore: true}

	result := NewPaginatedResult(items, info)

	assert.Equal(t, items, result.Items)
	assert.Equal(t, *info, result.Pagination)
}

func TestPaginationConstants(t *testing.T) {
	assert.Equal(t, 50, DefaultLimit)
	assert.Equal(t, 500, MaxLimit)
	assert.True(t, MaxLimit > DefaultLimit)
}

func TestPaginateSlice_IntSlice(t *testing.T) {
	items := []int{1, 2, 3, 4, 5}
	params := &PaginationParams{Limit: 2, Cursor: ""}

	result, info := PaginateSlice(items, params)

	assert.Equal(t, []int{1, 2}, result)
	assert.Equal(t, 5, info.Total)
	assert.True(t, info.HasMore)
}

func TestPaginateSlice_StructSlice(t *testing.T) {
	type Item struct {
		Name string
	}
	items := []Item{{Name: "a"}, {Name: "b"}, {Name: "c"}}
	params := &PaginationParams{Limit: 2, Cursor: ""}

	result, info := PaginateSlice(items, params)

	assert.Len(t, result, 2)
	assert.Equal(t, "a", result[0].Name)
	assert.Equal(t, 3, info.Total)
}

func TestCursorRoundTrip(t *testing.T) {
	// Test that encoding and decoding preserves values.
	tests := []struct {
		offset int
		query  string
	}{
		{0, ""},
		{100, "test"},
		{999999, "complex query with spaces"},
		{1, ""},
	}

	for _, tt := range tests {
		encoded := EncodeCursor(tt.offset, tt.query)
		decoded, err := DecodeCursor(encoded)

		require.NoError(t, err)
		assert.Equal(t, tt.offset, decoded.Offset)
		assert.Equal(t, tt.query, decoded.Query)
	}
}

func TestPaginateSlice_CursorPreservesQuery(t *testing.T) {
	items := []string{"a", "b", "c", "d", "e"}
	cursor := EncodeCursor(0, "myquery")
	params := &PaginationParams{Limit: 2, Cursor: cursor}

	_, info := PaginateSlice(items, params)

	// Decode the next cursor to verify query is preserved.
	nextCursor, err := DecodeCursor(info.NextCursor)
	require.NoError(t, err)
	assert.Equal(t, "myquery", nextCursor.Query)
}
