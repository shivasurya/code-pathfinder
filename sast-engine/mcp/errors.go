package mcp

import (
	"encoding/json"
	"fmt"
)

// Standard JSON-RPC 2.0 error codes.
const (
	ErrCodeParseError     = -32700
	ErrCodeInvalidRequest = -32600
	ErrCodeMethodNotFound = -32601
	ErrCodeInvalidParams  = -32602
	ErrCodeInternalError  = -32603

	// Custom server error codes (-32000 to -32099).
	ErrCodeSymbolNotFound   = -32001
	ErrCodeIndexNotReady    = -32002
	ErrCodeQueryTimeout     = -32003
	ErrCodeResultsTruncated = -32004
)

// errorMessages maps error codes to default messages.
var errorMessages = map[int]string{
	ErrCodeParseError:       "Parse error",
	ErrCodeInvalidRequest:   "Invalid Request",
	ErrCodeMethodNotFound:   "Method not found",
	ErrCodeInvalidParams:    "Invalid params",
	ErrCodeInternalError:    "Internal error",
	ErrCodeSymbolNotFound:   "Symbol not found",
	ErrCodeIndexNotReady:    "Index not ready",
	ErrCodeQueryTimeout:     "Query timeout",
	ErrCodeResultsTruncated: "Results truncated",
}

// Error implements the error interface for RPCError.
func (e *RPCError) Error() string {
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// NewRPCError creates a new RPC error with optional data.
func NewRPCError(code int, data any) *RPCError {
	message := errorMessages[code]
	if message == "" {
		message = "Unknown error"
	}
	return &RPCError{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

// NewRPCErrorWithMessage creates an RPC error with custom message.
func NewRPCErrorWithMessage(code int, message string, data any) *RPCError {
	return &RPCError{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

// ParseError creates a parse error response.
func ParseError(detail string) *RPCError {
	return NewRPCErrorWithMessage(ErrCodeParseError, "Parse error: "+detail, nil)
}

// InvalidRequestError creates an invalid request error.
func InvalidRequestError(detail string) *RPCError {
	return NewRPCErrorWithMessage(ErrCodeInvalidRequest, "Invalid request: "+detail, nil)
}

// MethodNotFoundError creates a method not found error.
func MethodNotFoundError(method string) *RPCError {
	return NewRPCErrorWithMessage(ErrCodeMethodNotFound,
		fmt.Sprintf("Method not found: %s", method),
		map[string]string{"method": method})
}

// InvalidParamsError creates an invalid params error.
func InvalidParamsError(detail string) *RPCError {
	return NewRPCErrorWithMessage(ErrCodeInvalidParams, "Invalid params: "+detail, nil)
}

// InternalError creates an internal error.
func InternalError(detail string) *RPCError {
	return NewRPCErrorWithMessage(ErrCodeInternalError, "Internal error: "+detail, nil)
}

// SymbolNotFoundError creates a symbol not found error with suggestions.
func SymbolNotFoundError(symbol string, suggestions []string) *RPCError {
	data := map[string]any{
		"symbol": symbol,
	}
	if len(suggestions) > 0 {
		data["suggestions"] = suggestions
	}
	return NewRPCErrorWithMessage(ErrCodeSymbolNotFound,
		fmt.Sprintf("Symbol not found: %s", symbol), data)
}

// IndexNotReadyError creates an index not ready error with optional progress info.
func IndexNotReadyError(phase string, progress float64) *RPCError {
	data := map[string]any{
		"phase":    phase,
		"progress": progress,
	}
	return NewRPCErrorWithMessage(ErrCodeIndexNotReady,
		fmt.Sprintf("Index not ready: %s (%.0f%% complete)", phase, progress*100),
		data)
}

// QueryTimeoutError creates a query timeout error.
func QueryTimeoutError(timeout string) *RPCError {
	return NewRPCErrorWithMessage(ErrCodeQueryTimeout,
		fmt.Sprintf("Query timed out after %s", timeout),
		map[string]string{"timeout": timeout})
}

// MakeErrorResponse creates a JSON-RPC error response from an RPCError.
func MakeErrorResponse(id any, err *RPCError) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   err,
	}
}

// ToolError represents a structured tool error response.
type ToolError struct {
	Error   string `json:"error"`
	Code    int    `json:"code,omitempty"`
	Details any    `json:"details,omitempty"`
}

// NewToolError creates a JSON-formatted tool error response.
func NewToolError(message string, code int, details any) string {
	te := ToolError{
		Error:   message,
		Code:    code,
		Details: details,
	}
	bytes, _ := json.Marshal(te)
	return string(bytes)
}

// ValidateRequiredParams checks for required parameters.
func ValidateRequiredParams(args map[string]any, required []string) *RPCError {
	missing := []string{}
	for _, param := range required {
		if _, ok := args[param]; !ok {
			missing = append(missing, param)
		}
	}
	if len(missing) > 0 {
		return InvalidParamsError(fmt.Sprintf("missing required parameters: %v", missing))
	}
	return nil
}

// ValidateStringParam validates a string parameter.
func ValidateStringParam(args map[string]any, name string) (string, *RPCError) {
	val, ok := args[name]
	if !ok {
		return "", InvalidParamsError(fmt.Sprintf("missing required parameter: %s", name))
	}
	str, ok := val.(string)
	if !ok {
		return "", InvalidParamsError(fmt.Sprintf("parameter %s must be a string", name))
	}
	if str == "" {
		return "", InvalidParamsError(fmt.Sprintf("parameter %s cannot be empty", name))
	}
	return str, nil
}

// ValidateIntParam validates an integer parameter with optional default.
func ValidateIntParam(args map[string]any, name string, defaultVal int) (int, *RPCError) {
	val, ok := args[name]
	if !ok {
		return defaultVal, nil
	}

	// JSON numbers come as float64.
	switch v := val.(type) {
	case float64:
		return int(v), nil
	case int:
		return v, nil
	default:
		return 0, InvalidParamsError(fmt.Sprintf("parameter %s must be a number", name))
	}
}
