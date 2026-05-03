package taint

// builtinTaintTransparent maps fully qualified function names to the parameter
// indices whose taint propagates to the return value.
//
// These are hardcoded summaries for stdlib functions whose behavior is KNOWN:
// the function preserves taint from input to output. This eliminates the need
// to analyze their bodies (which aren't available in user source code).
//
// Key: function FQN (after type enrichment, e.g., "fmt.Sprintf").
// Value: param indices that propagate taint to return (-1 = ALL params).
var builtinTaintTransparent = map[string][]int{
	// reflect — wraps values, preserving taint
	"reflect.ValueOf":         {0},
	"reflect.Value.String":    {}, // receiver propagates
	"reflect.Value.Interface": {},
	"reflect.Value.Bytes":     {},
	"reflect.Value.Int":       {},
	"reflect.Value.Elem":      {},

	// fmt — formatting propagates all args to output
	"fmt.Sprintf": {-1}, // ALL params → return
	"fmt.Fprintf": {-1}, // ALL params → written to writer
	"fmt.Errorf":  {-1}, // ALL params → error string

	// strings — transformation preserves taint
	"strings.Replace":    {0},
	"strings.ReplaceAll": {0},
	"strings.ToLower":    {0},
	"strings.ToUpper":    {0},
	"strings.TrimSpace":  {0},
	"strings.Trim":       {0},
	"strings.TrimLeft":   {0},
	"strings.TrimRight":  {0},
	"strings.TrimPrefix": {0},
	"strings.TrimSuffix": {0},
	"strings.Join":       {0}, // slice of strings → joined string
	"strings.Repeat":     {0},
	"strings.Map":        {1}, // mapping function + string

	// context — value propagation (middleware pattern)
	"context.WithValue":       {0, 2}, // parent ctx + value → new ctx carries both
	"context.Context.Value":   {},     // receiver (ctx) propagates taint to return

	// encoding
	"encoding/base64.StdEncoding.EncodeToString": {0},
	"encoding/base64.StdEncoding.DecodeString":   {0},
	"encoding/hex.EncodeToString":                {0},
	"encoding/hex.DecodeString":                  {0},

	// net/url
	"net/url.QueryEscape":   {0},
	"net/url.QueryUnescape": {0},
	"net/url.PathEscape":    {0},
	"net/url.PathUnescape":  {0},
}

// IsBuiltinTaintTransparent checks if a function has a hardcoded taint summary.
// Returns the param indices that propagate taint, or nil if not found.
// -1 in the slice means ALL parameters propagate.
func IsBuiltinTaintTransparent(funcFQN string) ([]int, bool) {
	params, ok := builtinTaintTransparent[funcFQN]
	return params, ok
}
