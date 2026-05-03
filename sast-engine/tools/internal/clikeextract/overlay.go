package clikeextract

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"gopkg.in/yaml.v3"
)

// Overlay is the parsed shape of c_stdlib_overlay.yaml / cpp_stdlib_overlay.yaml.
// It is a small hand-curated patch applied on top of tree-sitter extraction to
// fill in cases the parser cannot capture: template specialisations, missing
// __attribute__((format)) annotations, security tags, and cross-platform aliases.
//
// Schema (yaml.v3 tag conventions, lowercase field names matching the YAML keys):
//
//	schema_version: "1.0.0"     // string
//	language:       "c" | "cpp"
//	overrides:                  // each entry MUST specify exactly one of
//	                            //   {function} OR {class+method} OR {typedef}
//	                            //   OR {constant}
//	  - header:    "<string>"
//	    function:  "<name>"
//	    class:     "<fqn>"
//	    method:    "<name>"
//	    typedef:   "<name>"
//	    constant:  "<name>"
//	    return_type: "<type>"
//	    params:    [ {name, type, attribute?, required?}, ... ]
//	    confidence: 1.0          // optional, default 1.0
//	    security_tag: "<tag>"    // optional
//	    attribute:  "<attr>"     // optional
//	    throws:     "<exception>"// optional
//	    type:       "<type>"     // for typedef/constant entries
//	    value:      "<literal>"  // for constant entries
//	    note:       "<comment>"  // optional, ignored at runtime
//	cross_platform_aliases:
//	  - alias:     "<header>"
//	    canonical: "<header>"
//	skip:
//	  - prefix: "<...>"
//	  - exact:  "<...>"
//nolint:tagliatelle // YAML keys are snake_case to match hand-edited overlay files.
type Overlay struct {
	SchemaVersion string `yaml:"schema_version"`
	Language      string `yaml:"language"`

	Overrides            []OverlayOverride `yaml:"overrides"`
	CrossPlatformAliases []OverlayAlias    `yaml:"cross_platform_aliases"`
	Skip                 []OverlaySkip     `yaml:"skip"`

	// path is the file the overlay was loaded from; used for richer error
	// messages on validation failure.
	path string `yaml:"-"` //nolint:unused // populated by Load, surfaced in errors
}

// OverlayOverride describes one hand-curated override or insertion. The kind
// of entry is determined by which mutually-exclusive identifier fields are
// populated (Function / Method / Typedef / Constant). Validation enforces
// exactly-one.
//
//nolint:tagliatelle // YAML keys are snake_case to match hand-edited overlay files.
type OverlayOverride struct {
	Header     string         `yaml:"header"`
	Function   string         `yaml:"function,omitempty"`
	Class      string         `yaml:"class,omitempty"`
	Method     string         `yaml:"method,omitempty"`
	Typedef    string         `yaml:"typedef,omitempty"`
	Constant   string         `yaml:"constant,omitempty"`
	ReturnType string         `yaml:"return_type,omitempty"`
	Params     []OverlayParam `yaml:"params,omitempty"`

	// Confidence has no yaml omitempty marker because zero is a meaningful
	// signal — the merger replaces zero with 1.0 since hand-curated entries
	// are by definition high-confidence.
	Confidence  float32 `yaml:"confidence"`
	SecurityTag string  `yaml:"security_tag,omitempty"`
	Attribute   string  `yaml:"attribute,omitempty"`
	Throws      string  `yaml:"throws,omitempty"`
	Type        string  `yaml:"type,omitempty"`  // for typedef / constant entries
	Value       string  `yaml:"value,omitempty"` // for constant entries
	Note        string  `yaml:"note,omitempty"`  // ignored at runtime; for human readers
}

// OverlayParam is a parameter override entry. Required defaults to true (most
// overlay entries are well-formed signatures) — set explicitly to false for
// optional-after-default-args cases.
type OverlayParam struct {
	Name      string `yaml:"name"`
	Type      string `yaml:"type"`
	Attribute string `yaml:"attribute,omitempty"`
	Required  *bool  `yaml:"required,omitempty"`
}

// OverlayAlias maps a non-canonical header name to its canonical form. Phase 2
// stores these for later use; aliasing across platforms is a Phase 3 concern.
type OverlayAlias struct {
	Alias     string   `yaml:"alias"`
	Canonical string   `yaml:"canonical"`
	Platforms []string `yaml:"platforms,omitempty"`
}

// OverlaySkip names a single skip rule. Exactly one of Prefix or Exact must be
// set; both being empty is a validation error.
type OverlaySkip struct {
	Prefix string `yaml:"prefix,omitempty"`
	Exact  string `yaml:"exact,omitempty"`
}

// LoadOverlay reads, parses, and validates the YAML overlay at path. Returns nil
// (no overlay) if path is empty — callers may use this no-overlay shape to opt
// out, in which case every extracted entry's Source stays "header".
//
// Mismatch between the overlay's declared language and the wantLanguage argument
// is a hard error: a cpp overlay applied to a C extraction would silently inject
// invalid entries (e.g. class-method overrides into a C registry).
func LoadOverlay(path, wantLanguage string) (*Overlay, error) {
	if path == "" {
		return nil, nil //nolint:nilnil // empty path is intentional opt-out, not an error
	}
	data, err := os.ReadFile(path) //nolint:gosec // path comes from operator CLI flag, not user input
	if err != nil {
		return nil, fmt.Errorf("LoadOverlay: reading %q: %w", path, err)
	}

	var o Overlay
	if err := yaml.Unmarshal(data, &o); err != nil {
		return nil, fmt.Errorf("LoadOverlay: parsing YAML at %q: %w", path, err)
	}
	o.path = path

	if err := o.validate(wantLanguage); err != nil {
		return nil, fmt.Errorf("LoadOverlay: validating %q: %w", path, err)
	}
	return &o, nil
}

// validate enforces the overlay invariants declared in the package docs. The
// errors returned identify the offending entry by index so a hand-editor can
// jump straight to the line.
func (o *Overlay) validate(wantLanguage string) error {
	if o.SchemaVersion == "" {
		return errors.New("schema_version is required")
	}
	if o.Language == "" {
		return errors.New("language is required")
	}
	if o.Language != wantLanguage {
		return fmt.Errorf("overlay declares language=%q but extractor wants %q", o.Language, wantLanguage)
	}
	if o.Language != core.LanguageC && o.Language != core.LanguageCpp {
		return fmt.Errorf("language must be %q or %q (got %q)", core.LanguageC, core.LanguageCpp, o.Language)
	}

	for i, ov := range o.Overrides {
		if ov.Header == "" {
			return fmt.Errorf("overrides[%d]: header is required", i)
		}
		k, kerr := overrideKind(ov)
		if kerr != nil {
			return fmt.Errorf("overrides[%d]: %w", i, kerr)
		}
		if k == overrideMethod && ov.Class == "" {
			return fmt.Errorf("overrides[%d]: method override requires class", i)
		}
		if k == overrideMethod && o.Language == core.LanguageC {
			return fmt.Errorf("overrides[%d]: class+method only valid in cpp overlay", i)
		}
	}
	for i, sk := range o.Skip {
		if (sk.Prefix == "") == (sk.Exact == "") {
			return fmt.Errorf("skip[%d]: exactly one of prefix or exact must be set", i)
		}
	}
	for i, al := range o.CrossPlatformAliases {
		if al.Alias == "" || al.Canonical == "" {
			return fmt.Errorf("cross_platform_aliases[%d]: alias and canonical are both required", i)
		}
	}
	return nil
}

// overrideKind classifies a single override entry by which identifier fields
// are set. Returns an error if zero or more than one identifier is present.
func overrideKind(ov OverlayOverride) (overrideEntryKind, error) {
	count := 0
	var k overrideEntryKind
	if ov.Function != "" {
		count++
		k = overrideFunction
	}
	if ov.Method != "" {
		count++
		k = overrideMethod
	}
	if ov.Typedef != "" {
		count++
		k = overrideTypedef
	}
	if ov.Constant != "" {
		count++
		k = overrideConstant
	}
	if count == 0 {
		return overrideUnknown, errors.New("must specify exactly one of function, method, typedef, constant")
	}
	if count > 1 {
		return overrideUnknown, errors.New("must specify exactly one of function, method, typedef, constant (got multiple)")
	}
	return k, nil
}

type overrideEntryKind int

const (
	overrideUnknown overrideEntryKind = iota
	overrideFunction
	overrideMethod
	overrideTypedef
	overrideConstant
)

// MergeOverlay applies the overlay to one extracted CStdlibHeader, returning a
// (possibly modified) header. The returned pointer is the same as the input —
// merge happens in place. Counts the overlay entries actually applied via the
// returned int (used by the emitter to populate Statistics.OverlayOverrides).
//
// Merge rules:
//
//   - For each override whose Header matches the input, locate the target
//     symbol map (Functions / FreeFunctions / Classes.Methods / Typedefs /
//     Constants) and apply.
//   - If the target entry already exists from extraction, the overlay's values
//     replace the matching fields in-place; Source becomes "merged".
//   - If the target entry does not exist, a new one is inserted with
//     Source="overlay".
//   - Skip rules run last, AFTER overrides, so an override on a name that is
//     subsequently skipped is silently dropped (matches the user expectation
//     of "skip applies regardless of source").
//
// Returns the count of overlay entries that produced or refined a header entry
// (i.e. entries whose Header field matched the input header). Cross-platform
// aliases and skip rules do not count toward this number.
func MergeOverlay(h *core.CStdlibHeader, overlay *Overlay) int {
	if overlay == nil || h == nil {
		return 0
	}

	applied := 0
	for _, ov := range overlay.Overrides {
		if ov.Header != h.Header {
			continue
		}
		k, _ := overrideKind(ov) // pre-validated by Load
		switch k {
		case overrideFunction:
			applyFunctionOverride(h, ov)
		case overrideMethod:
			applyMethodOverride(h, ov)
		case overrideTypedef:
			applyTypedefOverride(h, ov)
		case overrideConstant:
			applyConstantOverride(h, ov)
		case overrideUnknown:
			continue
		}
		applied++
	}

	applySkipRules(h, overlay.Skip)
	return applied
}

func applyFunctionOverride(h *core.CStdlibHeader, ov OverlayOverride) {
	conf := ov.Confidence
	if conf == 0 {
		conf = 1.0
	}
	target := h.Functions
	if target == nil {
		target = make(map[string]*core.CStdlibFunction)
		h.Functions = target
	}

	existing, found := target[ov.Function]
	if !found {
		target[ov.Function] = &core.CStdlibFunction{
			FQN:         buildFunctionFQN(h, ov.Function),
			ReturnType:  ov.ReturnType,
			Params:      paramListFromOverlay(ov.Params),
			Confidence:  conf,
			Source:      core.SourceOverlay,
			SecurityTag: ov.SecurityTag,
			Attribute:   ov.Attribute,
			Throws:      ov.Throws,
		}
		return
	}
	if ov.ReturnType != "" {
		existing.ReturnType = ov.ReturnType
	}
	if len(ov.Params) > 0 {
		existing.Params = paramListFromOverlay(ov.Params)
	}
	if ov.SecurityTag != "" {
		existing.SecurityTag = ov.SecurityTag
	}
	if ov.Attribute != "" {
		existing.Attribute = ov.Attribute
	}
	if ov.Throws != "" {
		existing.Throws = ov.Throws
	}
	existing.Confidence = conf
	existing.Source = core.SourceMerged
}

func applyMethodOverride(h *core.CStdlibHeader, ov OverlayOverride) {
	conf := ov.Confidence
	if conf == 0 {
		conf = 1.0
	}
	if h.Classes == nil {
		h.Classes = make(map[string]*core.CppStdlibClass)
	}
	cls, ok := h.Classes[ov.Class]
	if !ok {
		cls = core.NewCppStdlibClass(ov.Class)
		h.Classes[ov.Class] = cls
	}

	existing, found := cls.Methods[ov.Method]
	if !found {
		cls.Methods[ov.Method] = &core.CStdlibFunction{
			FQN:        ov.Class + "::" + ov.Method,
			ReturnType: ov.ReturnType,
			Params:     paramListFromOverlay(ov.Params),
			Confidence: conf,
			Source:     core.SourceOverlay,
			Attribute:  ov.Attribute,
			Throws:     ov.Throws,
		}
		return
	}
	if ov.ReturnType != "" {
		existing.ReturnType = ov.ReturnType
	}
	if len(ov.Params) > 0 {
		existing.Params = paramListFromOverlay(ov.Params)
	}
	if ov.Attribute != "" {
		existing.Attribute = ov.Attribute
	}
	if ov.Throws != "" {
		existing.Throws = ov.Throws
	}
	existing.Confidence = conf
	existing.Source = core.SourceMerged
}

func applyTypedefOverride(h *core.CStdlibHeader, ov OverlayOverride) {
	if h.Typedefs == nil {
		h.Typedefs = make(map[string]*core.CStdlibTypedef)
	}
	if existing, ok := h.Typedefs[ov.Typedef]; ok {
		if ov.Type != "" {
			existing.Type = ov.Type
		}
		existing.Source = core.SourceMerged
		return
	}
	h.Typedefs[ov.Typedef] = &core.CStdlibTypedef{
		Type:   ov.Type,
		Source: core.SourceOverlay,
	}
}

func applyConstantOverride(h *core.CStdlibHeader, ov OverlayOverride) {
	if h.Constants == nil {
		h.Constants = make(map[string]*core.CStdlibConstant)
	}
	if existing, ok := h.Constants[ov.Constant]; ok {
		if ov.Type != "" {
			existing.Type = ov.Type
		}
		if ov.Value != "" {
			existing.Value = ov.Value
		}
		existing.Source = core.SourceMerged
		return
	}
	h.Constants[ov.Constant] = &core.CStdlibConstant{
		Type:   ov.Type,
		Value:  ov.Value,
		Source: core.SourceOverlay,
	}
}

// applySkipRules removes any function / typedef / constant whose name matches
// a skip rule. C++ class methods are not currently subject to skip rules
// (they are scoped under a class, so name collisions are unlikely); add as a
// follow-up if a real-world overlay needs it.
func applySkipRules(h *core.CStdlibHeader, skips []OverlaySkip) {
	if len(skips) == 0 {
		return
	}
	for name := range h.Functions {
		if matchesAnySkip(name, skips) {
			delete(h.Functions, name)
		}
	}
	for name := range h.FreeFunctions {
		if matchesAnySkip(name, skips) {
			delete(h.FreeFunctions, name)
		}
	}
	for name := range h.Typedefs {
		if matchesAnySkip(name, skips) {
			delete(h.Typedefs, name)
		}
	}
	for name := range h.Constants {
		if matchesAnySkip(name, skips) {
			delete(h.Constants, name)
		}
	}
}

func matchesAnySkip(name string, skips []OverlaySkip) bool {
	for _, sk := range skips {
		if sk.Prefix != "" && strings.HasPrefix(name, sk.Prefix) {
			return true
		}
		if sk.Exact != "" && name == sk.Exact {
			return true
		}
	}
	return false
}

// paramListFromOverlay converts overlay-shaped params into the public schema
// params, defaulting Required to true when the overlay author left it blank.
func paramListFromOverlay(ps []OverlayParam) []*core.CStdlibParam {
	if len(ps) == 0 {
		return []*core.CStdlibParam{}
	}
	out := make([]*core.CStdlibParam, 0, len(ps))
	for _, p := range ps {
		req := true
		if p.Required != nil {
			req = *p.Required
		}
		out = append(out, &core.CStdlibParam{
			Name:      p.Name,
			Type:      p.Type,
			Required:  req,
			Attribute: p.Attribute,
		})
	}
	return out
}

// buildFunctionFQN derives a fully-qualified name for a header-only entry, using
// the header's ModuleID if present (e.g. "c::stdio") or falling back to the
// header name if not. This keeps overlay-only entries consistent with extracted
// entries even when the overlay omits the FQN.
func buildFunctionFQN(h *core.CStdlibHeader, name string) string {
	if h.ModuleID != "" {
		return h.ModuleID + "::" + name
	}
	return name
}
