package core

// CStdlibLoader is the interface the C call-graph builder uses to query the
// stdlib registry. The PR-02 file:// loader and the upcoming PR-03 HTTP loader
// both satisfy it; tests substitute fakes.
//
// The interface deliberately keeps Logger out of the per-symbol accessors:
// LoadManifest is the one-shot startup operation that needs progress logging,
// and the per-symbol queries happen in tight resolver loops where logging
// would be noise.
type CStdlibLoader interface {
	LoadManifest(logger CStdlibLogger) error
	GetHeader(name string) (*CStdlibHeader, error)
	GetFunction(headerName, funcName string) (*CStdlibFunction, error)
	Platform() string
	HeaderCount() int
	// ListHeaders returns every header name in the loaded manifest in
	// deterministic order. The resolver uses this to fall back to a
	// global manifest scan when a call site doesn't directly #include
	// the header that owns the symbol — common with C++ codebases that
	// rely on transitive includes (vector pulling in utility, etc).
	//
	// Returns an empty slice when LoadManifest has not been called.
	ListHeaders() []string
}

// CppStdlibLoader extends CStdlibLoader with C++-specific accessors. Free
// functions like std::move that live in a namespace map are looked up via
// GetFreeFunction; class methods (vector::push_back) via GetMethod.
type CppStdlibLoader interface {
	CStdlibLoader
	GetClass(headerName, classFQN string) (*CppStdlibClass, error)
	GetMethod(headerName, classFQN, methodName string) (*CStdlibFunction, error)
	GetFreeFunction(headerName, fqn string) (*CStdlibFunction, error)
}

// CStdlibLogger is the subset of *output.Logger that loaders need. Defining a
// narrow interface here (rather than importing the full output package into
// core/) keeps the dependency direction clean — core has no upstream
// dependencies on output.
type CStdlibLogger interface {
	Debug(format string, args ...any)
	Statistic(format string, args ...any)
	Warning(format string, args ...any)
}

// CStdlibRegistry is the root in-memory container for C/C++ stdlib data on a single
// (platform, language) axis (e.g. linux/c, linux/cpp, windows/c). It is populated by
// the loader (PR-02) from registry JSON hosted on the CDN and consulted by the call
// graph builder when a call site cannot be resolved against project-internal symbols.
//
// One CStdlibRegistry instance is created per (platform, language) pair the analyzer
// touches; the engine typically holds two — one for C, one for C++ — for the
// auto-detected target platform.
type CStdlibRegistry struct {
	// Manifest lists the per-header files available on the CDN and the
	// aggregate statistics. Always non-nil after a successful Load.
	Manifest *CStdlibManifest
	// Headers maps header name (e.g. "stdio.h", "vector") to the parsed
	// per-header content. Populated lazily by the loader: an entry is
	// present only after the corresponding header file has been fetched.
	Headers map[string]*CStdlibHeader
}

// NewCStdlibRegistry creates an initialized registry with a pre-allocated headers map.
func NewCStdlibRegistry() *CStdlibRegistry {
	return &CStdlibRegistry{
		Headers: make(map[string]*CStdlibHeader),
	}
}

// GetHeader returns the parsed per-header content for the given header, or nil
// if the loader has not (yet) fetched it. Callers expecting on-demand fetch
// should go through the loader's lazy-fetch path rather than this accessor.
func (r *CStdlibRegistry) GetHeader(name string) *CStdlibHeader {
	return r.Headers[name]
}

// HasHeader reports whether the given header has been loaded into the registry.
func (r *CStdlibRegistry) HasHeader(name string) bool {
	_, ok := r.Headers[name]
	return ok
}

// HeaderCount returns the number of headers currently materialised in memory.
// This is at most len(Manifest.Headers); equal only after every header has been
// fetched (eager load) which is unusual — most projects include 10–30 stdlib
// headers out of the 80–110 available.
func (r *CStdlibRegistry) HeaderCount() int {
	return len(r.Headers)
}

// GetFunction is a convenience accessor: looks up the function by name within the
// given header. Returns nil if either the header is not loaded or the function is
// absent. Used by both C registries (top-level functions) and C++ registries (free
// functions in a namespace, indexed in the same map under their fully-qualified
// name e.g. "std::move").
func (r *CStdlibRegistry) GetFunction(headerName, funcName string) *CStdlibFunction {
	h := r.GetHeader(headerName)
	if h == nil {
		return nil
	}
	if f, ok := h.Functions[funcName]; ok {
		return f
	}
	return h.FreeFunctions[funcName]
}

// GetClass returns the C++ class metadata for the given header + class FQN,
// or nil if missing. Always returns nil for C-language registries.
func (r *CStdlibRegistry) GetClass(headerName, classFQN string) *CppStdlibClass {
	h := r.GetHeader(headerName)
	if h == nil {
		return nil
	}
	return h.Classes[classFQN]
}

// GetMethod is a two-step accessor: header → class → method. Returns nil if any
// step misses. Used by the C++ resolver when dispatching `obj.method()` against
// a stdlib type whose receiver was inferred earlier in the pipeline.
func (r *CStdlibRegistry) GetMethod(headerName, classFQN, methodName string) *CStdlibFunction {
	cls := r.GetClass(headerName, classFQN)
	if cls == nil {
		return nil
	}
	return cls.Methods[methodName]
}

// CStdlibManifest is the top-level manifest.json for a (platform, language) pair.
// It is the smallest file in the registry tree (typically <10 KB) and serves as the
// directory the loader uses to discover, validate, and lazily fetch per-header files.
//
//nolint:tagliatelle // JSON tags match the registry format on disk (snake_case).
type CStdlibManifest struct {
	SchemaVersion    string                `json:"schema_version"`
	RegistryVersion  string                `json:"registry_version"`
	Platform         string                `json:"platform"`
	Language         string                `json:"language"`
	SystemTag        string                `json:"system_tag"`
	GeneratedAt      string                `json:"generated_at"`
	GeneratorVersion string                `json:"generator_version"`
	BaseURL          string                `json:"base_url"`
	Headers          []*CStdlibHeaderEntry `json:"headers"`
	Statistics       *CStdlibStatistics    `json:"statistics"`
}

// NewCStdlibManifest creates an initialized manifest with a pre-allocated headers slice.
func NewCStdlibManifest() *CStdlibManifest {
	return &CStdlibManifest{
		Headers:    make([]*CStdlibHeaderEntry, 0),
		Statistics: &CStdlibStatistics{},
	}
}

// HasHeader reports whether the manifest has an entry for the given header name.
// Comparison is exact (case-sensitive) against the Header field; the C++ headers
// "vector"/"string"/"map" appear as-is, the C headers as "stdio.h"/"string.h", etc.
func (m *CStdlibManifest) HasHeader(name string) bool {
	return m.GetHeaderEntry(name) != nil
}

// GetHeaderEntry returns the manifest entry for a header, or nil if absent.
func (m *CStdlibManifest) GetHeaderEntry(name string) *CStdlibHeaderEntry {
	for _, e := range m.Headers {
		if e.Header == name {
			return e
		}
	}
	return nil
}

// CStdlibHeaderEntry is one row in the manifest's header list. It carries enough
// information for the loader to download, validate, and cache the per-header file.
//
//nolint:tagliatelle // JSON tags match the registry format on disk (snake_case).
type CStdlibHeaderEntry struct {
	Header   string `json:"header"`
	ModuleID string `json:"module_id"`
	File     string `json:"file"`
	URL      string `json:"url"`
	Size     int64  `json:"size_bytes"`
	Checksum string `json:"checksum"`
}

// CStdlibStatistics carries aggregate counts across all headers in the manifest.
// Useful for `pathfinder resolution-report` summary blocks and for end-of-generation
// sanity-checking ("are we anywhere near the budget?").
//
//nolint:tagliatelle // JSON tags match the registry format on disk (snake_case).
type CStdlibStatistics struct {
	TotalHeaders     int `json:"total_headers"`
	TotalFunctions   int `json:"total_functions"`
	TotalClasses     int `json:"total_classes,omitempty"`
	TotalTypedefs    int `json:"total_typedefs"`
	TotalConstants   int `json:"total_constants"`
	OverlayOverrides int `json:"overlay_overrides"`
}

// CStdlibHeader is the per-header registry file content. One file per stdlib header
// (e.g. stdio_stdlib.json, vector_stdlib.json) — chosen over per-module aggregation
// because it maps directly to the `#include <X>` directives the engine already tracks
// (see Phase 1 c_module.go BuildCIncludeMap).
//
// One type serves both C and C++. The C++ specific maps (Classes, FreeFunctions,
// Namespaces) are tagged omitempty so the C variants emit clean output without them.
//
//nolint:tagliatelle // JSON tags match the registry format on disk (snake_case).
type CStdlibHeader struct {
	SchemaVersion string                      `json:"schema_version"`
	Header        string                      `json:"header"`
	ModuleID      string                      `json:"module_id"`
	Language      string                      `json:"language"`
	Platform      string                      `json:"platform"`
	SystemTag     string                      `json:"system_tag"`
	GeneratedAt   string                      `json:"generated_at"`
	Functions     map[string]*CStdlibFunction `json:"functions,omitempty"`
	Typedefs      map[string]*CStdlibTypedef  `json:"typedefs,omitempty"`
	Constants     map[string]*CStdlibConstant `json:"constants,omitempty"`
	Classes       map[string]*CppStdlibClass  `json:"classes,omitempty"`
	Namespaces    []string                    `json:"namespaces,omitempty"`
	FreeFunctions map[string]*CStdlibFunction `json:"free_functions,omitempty"`
}

// NewCStdlibHeader creates an initialized header with all maps pre-allocated. The
// caller is expected to set SchemaVersion, Header, ModuleID, Language, Platform,
// SystemTag, and GeneratedAt before populating the symbol maps.
func NewCStdlibHeader() *CStdlibHeader {
	return &CStdlibHeader{
		Functions:     make(map[string]*CStdlibFunction),
		Typedefs:      make(map[string]*CStdlibTypedef),
		Constants:     make(map[string]*CStdlibConstant),
		Classes:       make(map[string]*CppStdlibClass),
		FreeFunctions: make(map[string]*CStdlibFunction),
	}
}

// CStdlibFunction is the in-memory record for a C function, a C++ free function, or
// a C++ class method. The shape is identical for all three because every consumer
// (resolver, security rule, type-info propagator) wants the same fields. The FQN
// distinguishes the call form: bare ("printf"), namespaced ("std::move"), or
// class-qualified ("std::vector::push_back").
//
//nolint:tagliatelle // JSON tags match the registry format on disk (snake_case).
type CStdlibFunction struct {
	FQN         string          `json:"fqn"`
	ReturnType  string          `json:"return_type"`
	Params      []*CStdlibParam `json:"params"`
	Confidence  float32         `json:"confidence"`
	Source      string          `json:"source"`
	SecurityTag string          `json:"security_tag,omitempty"`
	Attribute   string          `json:"attribute,omitempty"`
	Throws      string          `json:"throws,omitempty"`
}

// CStdlibParam is a single parameter in a function signature. Variadic positions
// are encoded as a synthetic param with Name="..." and Type="variadic" — the engine
// recognises this convention and skips type checking against variadic positions.
//
//nolint:tagliatelle // JSON tags match the registry format on disk (snake_case).
type CStdlibParam struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Required  bool   `json:"required"`
	Attribute string `json:"attribute,omitempty"`
}

// CStdlibTypedef is a single typedef entry. PlatformSpecific is set when the
// underlying type changes across platforms (e.g. size_t differs between glibc and
// MSVCRT) — the loader uses this to decide whether to surface platform-specific
// values when the analyzer is run against a single-platform build.
//
//nolint:tagliatelle // JSON tags match the registry format on disk (snake_case).
type CStdlibTypedef struct {
	Type             string `json:"type"`
	PlatformSpecific bool   `json:"platform_specific"`
	Source           string `json:"source"`
}

// CStdlibConstant is a #define or const-expression value extracted from a header.
// Value is the literal text from the header (e.g. "-1" for EOF, "8192" for BUFSIZ);
// it may be empty for symbolic constants whose body is not a single literal.
//
//nolint:tagliatelle // JSON tags match the registry format on disk (snake_case).
type CStdlibConstant struct {
	Type   string `json:"type"`
	Value  string `json:"value,omitempty"`
	Source string `json:"source"`
}

// CppStdlibClass is the in-memory record for a C++ class or class template
// (e.g. std::vector, std::basic_string). TypeParams carries the template parameter
// names declared on the class — the resolver uses them to substitute concrete types
// into method return types ("T&" with receiver `std::vector<int>` → "int&").
//
//nolint:tagliatelle // JSON tags match the registry format on disk (snake_case).
type CppStdlibClass struct {
	FQN                 string                      `json:"fqn"`
	TypeParams          []string                    `json:"type_params,omitempty"`
	DefaultTemplateArgs map[string]string           `json:"default_template_args,omitempty"`
	Methods             map[string]*CStdlibFunction `json:"methods"`
	Constructors        []*CppStdlibConstructor     `json:"constructors,omitempty"`
}

// NewCppStdlibClass creates an initialized class with a pre-allocated methods map.
func NewCppStdlibClass(fqn string) *CppStdlibClass {
	return &CppStdlibClass{
		FQN:                 fqn,
		DefaultTemplateArgs: make(map[string]string),
		Methods:             make(map[string]*CStdlibFunction),
	}
}

// CppStdlibConstructor is one constructor overload. Constructors share Params and
// Source semantics with CStdlibFunction but have no return type, no throws annotation,
// and no FQN of their own (the FQN is the class's, by convention).
//
//nolint:tagliatelle // JSON tags match the registry format on disk (snake_case).
type CppStdlibConstructor struct {
	Params []*CStdlibParam `json:"params"`
	Source string          `json:"source"`
}

// Source-field constants. The Source field on any extracted entry must be exactly
// one of these three values. The generator (PR-01) sets it during extract+merge;
// the resolver (PR-02) uses it as a confidence signal alongside the Confidence
// float (header < merged < overlay, in terms of curator-vetted reliability).
const (
	// SourceHeader marks an entry that came purely from tree-sitter parsing.
	SourceHeader = "header"
	// SourceOverlay marks an entry that exists only in the YAML overlay — no
	// matching header-extracted entry was present.
	SourceOverlay = "overlay"
	// SourceMerged marks an entry that was present in both the header parse and
	// the overlay; the overlay's values won on every conflicting field.
	SourceMerged = "merged"
)

// Language constants, used in CStdlibManifest.Language and CStdlibHeader.Language.
const (
	LanguageC   = "c"
	LanguageCpp = "cpp"
)

// Platform constants, used in CStdlibManifest.Platform and CStdlibHeader.Platform.
const (
	PlatformLinux   = "linux"
	PlatformWindows = "windows"
	PlatformDarwin  = "darwin"
)
