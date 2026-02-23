package mcp

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRPCError_Error(t *testing.T) {
	err := &RPCError{Code: -32600, Message: "Invalid Request"}
	assert.Equal(t, "[-32600] Invalid Request", err.Error())
}

func TestNewRPCError(t *testing.T) {
	err := NewRPCError(ErrCodeParseError, nil)

	assert.Equal(t, ErrCodeParseError, err.Code)
	assert.Equal(t, "Parse error", err.Message)
	assert.Nil(t, err.Data)
}

func TestNewRPCError_WithData(t *testing.T) {
	data := map[string]string{"detail": "unexpected EOF"}
	err := NewRPCError(ErrCodeParseError, data)

	assert.Equal(t, data, err.Data)
}

func TestNewRPCError_UnknownCode(t *testing.T) {
	err := NewRPCError(-99999, nil)
	assert.Equal(t, "Unknown error", err.Message)
}

func TestNewRPCErrorWithMessage(t *testing.T) {
	err := NewRPCErrorWithMessage(-32600, "Custom message", nil)

	assert.Equal(t, -32600, err.Code)
	assert.Equal(t, "Custom message", err.Message)
}

func TestParseError(t *testing.T) {
	err := ParseError("unexpected token")

	assert.Equal(t, ErrCodeParseError, err.Code)
	assert.Contains(t, err.Message, "unexpected token")
}

func TestInvalidRequestError(t *testing.T) {
	err := InvalidRequestError("missing jsonrpc field")

	assert.Equal(t, ErrCodeInvalidRequest, err.Code)
	assert.Contains(t, err.Message, "missing jsonrpc field")
}

func TestMethodNotFoundError(t *testing.T) {
	err := MethodNotFoundError("unknown/method")

	assert.Equal(t, ErrCodeMethodNotFound, err.Code)
	assert.Contains(t, err.Message, "unknown/method")

	data := err.Data.(map[string]string)
	assert.Equal(t, "unknown/method", data["method"])
}

func TestInvalidParamsError(t *testing.T) {
	err := InvalidParamsError("symbol is required")

	assert.Equal(t, ErrCodeInvalidParams, err.Code)
	assert.Contains(t, err.Message, "symbol is required")
}

func TestInternalError(t *testing.T) {
	err := InternalError("database connection failed")

	assert.Equal(t, ErrCodeInternalError, err.Code)
	assert.Contains(t, err.Message, "database connection failed")
}

func TestSymbolNotFoundError(t *testing.T) {
	err := SymbolNotFoundError("MyClass", []string{"MyClass2", "MyClassHelper"})

	assert.Equal(t, ErrCodeSymbolNotFound, err.Code)
	assert.Contains(t, err.Message, "MyClass")

	data := err.Data.(map[string]any)
	assert.Equal(t, "MyClass", data["symbol"])
	assert.Contains(t, data["suggestions"], "MyClass2")
}

func TestSymbolNotFoundError_NoSuggestions(t *testing.T) {
	err := SymbolNotFoundError("xyz", nil)

	data := err.Data.(map[string]any)
	_, hasSuggestions := data["suggestions"]
	assert.False(t, hasSuggestions)
}

func TestIndexNotReadyError(t *testing.T) {
	err := IndexNotReadyError("parsing", 0.5)

	assert.Equal(t, ErrCodeIndexNotReady, err.Code)
	assert.Contains(t, err.Message, "parsing")
	assert.Contains(t, err.Message, "50%")

	data := err.Data.(map[string]any)
	assert.Equal(t, "parsing", data["phase"])
	assert.Equal(t, 0.5, data["progress"])
}

func TestQueryTimeoutError(t *testing.T) {
	err := QueryTimeoutError("30s")

	assert.Equal(t, ErrCodeQueryTimeout, err.Code)
	assert.Contains(t, err.Message, "30s")
}

func TestMakeErrorResponse(t *testing.T) {
	rpcErr := ParseError("bad json")
	resp := MakeErrorResponse(1, rpcErr)

	assert.Equal(t, "2.0", resp.JSONRPC)
	assert.Equal(t, 1, resp.ID)
	assert.Equal(t, rpcErr, resp.Error)
	assert.Nil(t, resp.Result)
}

func TestMakeErrorResponse_NilID(t *testing.T) {
	rpcErr := ParseError("bad json")
	resp := MakeErrorResponse(nil, rpcErr)

	assert.Equal(t, "2.0", resp.JSONRPC)
	assert.Nil(t, resp.ID)
	assert.Equal(t, rpcErr, resp.Error)
}

func TestNewToolError(t *testing.T) {
	result := NewToolError("Symbol not found", ErrCodeSymbolNotFound, map[string]string{"symbol": "foo"})

	var parsed ToolError
	err := json.Unmarshal([]byte(result), &parsed)
	require.NoError(t, err)

	assert.Equal(t, "Symbol not found", parsed.Error)
	assert.Equal(t, ErrCodeSymbolNotFound, parsed.Code)
}

func TestNewToolError_NoDetails(t *testing.T) {
	result := NewToolError("Internal error", ErrCodeInternalError, nil)

	var parsed ToolError
	err := json.Unmarshal([]byte(result), &parsed)
	require.NoError(t, err)

	assert.Equal(t, "Internal error", parsed.Error)
	assert.Nil(t, parsed.Details)
}

func TestValidateRequiredParams(t *testing.T) {
	args := map[string]any{
		"name": "test",
	}

	// Has required param.
	err := ValidateRequiredParams(args, []string{"name"})
	assert.Nil(t, err)

	// Missing required param.
	err = ValidateRequiredParams(args, []string{"name", "value"})
	assert.NotNil(t, err)
	assert.Equal(t, ErrCodeInvalidParams, err.Code)
	assert.Contains(t, err.Message, "value")
}

func TestValidateRequiredParams_AllPresent(t *testing.T) {
	args := map[string]any{
		"name":  "test",
		"value": 123,
	}

	err := ValidateRequiredParams(args, []string{"name", "value"})
	assert.Nil(t, err)
}

func TestValidateRequiredParams_Empty(t *testing.T) {
	args := map[string]any{}

	err := ValidateRequiredParams(args, []string{"name"})
	assert.NotNil(t, err)
	assert.Contains(t, err.Message, "name")
}

func TestValidateStringParam(t *testing.T) {
	args := map[string]any{
		"name":   "test",
		"number": 123,
		"empty":  "",
	}

	// Valid string.
	val, err := ValidateStringParam(args, "name")
	assert.Nil(t, err)
	assert.Equal(t, "test", val)

	// Missing param.
	_, err = ValidateStringParam(args, "missing")
	assert.NotNil(t, err)
	assert.Contains(t, err.Message, "missing")

	// Wrong type.
	_, err = ValidateStringParam(args, "number")
	assert.NotNil(t, err)
	assert.Contains(t, err.Message, "must be a string")

	// Empty string.
	_, err = ValidateStringParam(args, "empty")
	assert.NotNil(t, err)
	assert.Contains(t, err.Message, "cannot be empty")
}

func TestValidateIntParam(t *testing.T) {
	args := map[string]any{
		"count": float64(10),
		"limit": 100,
		"name":  "test",
	}

	// Valid float64 (from JSON).
	val, err := ValidateIntParam(args, "count", 5)
	assert.Nil(t, err)
	assert.Equal(t, 10, val)

	// Valid int.
	val, err = ValidateIntParam(args, "limit", 50)
	assert.Nil(t, err)
	assert.Equal(t, 100, val)

	// Missing - use default.
	val, err = ValidateIntParam(args, "missing", 25)
	assert.Nil(t, err)
	assert.Equal(t, 25, val)

	// Wrong type.
	_, err = ValidateIntParam(args, "name", 0)
	assert.NotNil(t, err)
	assert.Contains(t, err.Message, "must be a number")
}

func TestErrorCodes_AreCorrect(t *testing.T) {
	// Verify standard JSON-RPC 2.0 codes.
	assert.Equal(t, -32700, ErrCodeParseError)
	assert.Equal(t, -32600, ErrCodeInvalidRequest)
	assert.Equal(t, -32601, ErrCodeMethodNotFound)
	assert.Equal(t, -32602, ErrCodeInvalidParams)
	assert.Equal(t, -32603, ErrCodeInternalError)

	// Verify custom codes are in valid range.
	assert.True(t, ErrCodeSymbolNotFound >= -32099 && ErrCodeSymbolNotFound <= -32000)
	assert.True(t, ErrCodeIndexNotReady >= -32099 && ErrCodeIndexNotReady <= -32000)
	assert.True(t, ErrCodeQueryTimeout >= -32099 && ErrCodeQueryTimeout <= -32000)
	assert.True(t, ErrCodeResultsTruncated >= -32099 && ErrCodeResultsTruncated <= -32000)
}

func TestRPCError_JSONSerialization(t *testing.T) {
	err := SymbolNotFoundError("foo", []string{"foobar"})

	bytes, jsonErr := json.Marshal(err)
	require.NoError(t, jsonErr)

	var parsed RPCError
	jsonErr = json.Unmarshal(bytes, &parsed)
	require.NoError(t, jsonErr)

	assert.Equal(t, err.Code, parsed.Code)
	assert.Equal(t, err.Message, parsed.Message)
}

func TestAllErrorMessages_Defined(t *testing.T) {
	// Verify all error codes have messages.
	codes := []int{
		ErrCodeParseError,
		ErrCodeInvalidRequest,
		ErrCodeMethodNotFound,
		ErrCodeInvalidParams,
		ErrCodeInternalError,
		ErrCodeSymbolNotFound,
		ErrCodeIndexNotReady,
		ErrCodeQueryTimeout,
		ErrCodeResultsTruncated,
	}

	for _, code := range codes {
		msg := errorMessages[code]
		assert.NotEmpty(t, msg, "Error code %d should have a message", code)
	}
}
