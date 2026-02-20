package core

// GoStdlibRegistry is the root in-memory container for all stdlib package data of a specific Go version.
// It provides fast lookup of package metadata loaded from the CDN-hosted registry.
type GoStdlibRegistry struct {
	// Registry metadata and the ordered list of available packages.
	Manifest *GoManifest
	// Packages maps import path to package data (e.g., "fmt" → *GoStdlibPackage).
	Packages map[string]*GoStdlibPackage
}

// NewGoStdlibRegistry creates an initialized GoStdlibRegistry with a pre-allocated packages map.
func NewGoStdlibRegistry() *GoStdlibRegistry {
	return &GoStdlibRegistry{
		Packages: make(map[string]*GoStdlibPackage),
	}
}

// GetPackage returns the package data for the given import path.
// Returns nil if the package is not loaded.
func (r *GoStdlibRegistry) GetPackage(importPath string) *GoStdlibPackage {
	return r.Packages[importPath]
}

// HasPackage reports whether a package is loaded in the registry.
func (r *GoStdlibRegistry) HasPackage(importPath string) bool {
	_, exists := r.Packages[importPath]
	return exists
}

// GetFunction returns the function metadata for the given package and function name.
// Returns nil if the package is not found or the function does not exist in the package.
func (r *GoStdlibRegistry) GetFunction(importPath, funcName string) *GoStdlibFunction {
	pkg := r.GetPackage(importPath)
	if pkg == nil {
		return nil
	}
	return pkg.Functions[funcName]
}

// GetType returns the type metadata for the given package and type name.
// Returns nil if the package is not found or the type does not exist in the package.
func (r *GoStdlibRegistry) GetType(importPath, typeName string) *GoStdlibType {
	pkg := r.GetPackage(importPath)
	if pkg == nil {
		return nil
	}
	return pkg.Types[typeName]
}

// GetConstant returns the constant metadata for the given package and constant name.
// Returns nil if the package is not found or the constant does not exist in the package.
func (r *GoStdlibRegistry) GetConstant(importPath, constName string) *GoStdlibConstant {
	pkg := r.GetPackage(importPath)
	if pkg == nil {
		return nil
	}
	return pkg.Constants[constName]
}

// GetVariable returns the package-level variable metadata for the given package and variable name.
// Returns nil if the package is not found or the variable does not exist in the package.
func (r *GoStdlibRegistry) GetVariable(importPath, varName string) *GoStdlibVariable {
	pkg := r.GetPackage(importPath)
	if pkg == nil {
		return nil
	}
	return pkg.Variables[varName]
}

// PackageCount returns the total number of packages loaded in the registry.
func (r *GoStdlibRegistry) PackageCount() int {
	return len(r.Packages)
}

// GoManifest contains registry metadata and the list of available packages (manifest.json).
// It is the first file fetched from the CDN and acts as an index for lazy package loading.
//
//nolint:tagliatelle // JSON tags match Go stdlib registry format (snake_case).
type GoManifest struct {
	SchemaVersion    string            `json:"schema_version"`
	RegistryVersion  string            `json:"registry_version"`
	GoVersion        GoVersionInfo     `json:"go_version"`
	GeneratedAt      string            `json:"generated_at"`
	GeneratorVersion string            `json:"generator_version"`
	BaseURL          string            `json:"base_url"`
	Packages         []*GoPackageEntry `json:"packages"`
	Statistics       *GoRegistryStats  `json:"statistics"`
}

// NewGoManifest creates an initialized GoManifest with a pre-allocated packages slice.
func NewGoManifest() *GoManifest {
	return &GoManifest{
		Packages: make([]*GoPackageEntry, 0),
	}
}

// HasPackage reports whether a package entry with the given import path exists in the manifest.
func (m *GoManifest) HasPackage(importPath string) bool {
	for _, entry := range m.Packages {
		if entry.ImportPath == importPath {
			return true
		}
	}
	return false
}

// GetPackageEntry returns the package entry for the given import path.
// Returns nil if the import path is not found in the manifest.
func (m *GoManifest) GetPackageEntry(importPath string) *GoPackageEntry {
	for _, entry := range m.Packages {
		if entry.ImportPath == importPath {
			return entry
		}
	}
	return nil
}

// GoVersionInfo contains the Go release version details embedded in the manifest.
//
//nolint:tagliatelle // JSON tags match Go stdlib registry format (snake_case).
type GoVersionInfo struct {
	Major       int    `json:"major"`
	Minor       int    `json:"minor"`
	Patch       int    `json:"patch"`
	Full        string `json:"full"`
	ReleaseDate string `json:"release_date"`
}

// GoPackageEntry represents a single package's metadata entry inside the manifest.
// It carries enough information to validate and download the full package file.
//
//nolint:tagliatelle // JSON tags match Go stdlib registry format (snake_case).
type GoPackageEntry struct {
	ImportPath    string `json:"import_path"`
	Checksum      string `json:"checksum"`
	FileSize      int64  `json:"file_size"`
	FunctionCount int    `json:"function_count"`
	TypeCount     int    `json:"type_count"`
	ConstantCount int    `json:"constant_count"`
}

// GoRegistryStats contains aggregate statistics for the full registry across all packages.
//
//nolint:tagliatelle // JSON tags match Go stdlib registry format (snake_case).
type GoRegistryStats struct {
	TotalPackages        int `json:"total_packages"`
	TotalFunctions       int `json:"total_functions"`
	TotalTypes           int `json:"total_types"`
	TotalConstants       int `json:"total_constants"`
	PackagesWithGenerics int `json:"packages_with_generics"`
}

// GoStdlibPackage represents the complete exported API surface of a single stdlib package.
// It is stored as an individual JSON file (e.g., fmt_stdlib.json) on the CDN.
//
//nolint:tagliatelle // JSON tags match Go stdlib registry format (snake_case).
type GoStdlibPackage struct {
	ImportPath  string                       `json:"import_path"`
	GoVersion   string                       `json:"go_version"`
	GeneratedAt string                       `json:"generated_at"`
	Functions   map[string]*GoStdlibFunction `json:"functions"`
	Types       map[string]*GoStdlibType     `json:"types"`
	Constants   map[string]*GoStdlibConstant `json:"constants"`
	Variables   map[string]*GoStdlibVariable `json:"variables"`
}

// NewGoStdlibPackage creates an initialized GoStdlibPackage with all maps pre-allocated.
func NewGoStdlibPackage(importPath, goVersion string) *GoStdlibPackage {
	return &GoStdlibPackage{
		ImportPath: importPath,
		GoVersion:  goVersion,
		Functions:  make(map[string]*GoStdlibFunction),
		Types:      make(map[string]*GoStdlibType),
		Constants:  make(map[string]*GoStdlibConstant),
		Variables:  make(map[string]*GoStdlibVariable),
	}
}

// FunctionCount returns the number of exported functions declared in this package.
func (p *GoStdlibPackage) FunctionCount() int {
	return len(p.Functions)
}

// TypeCount returns the number of exported types declared in this package.
func (p *GoStdlibPackage) TypeCount() int {
	return len(p.Types)
}

// ConstantCount returns the number of exported constants declared in this package.
func (p *GoStdlibPackage) ConstantCount() int {
	return len(p.Constants)
}

// VariableCount returns the number of exported package-level variables declared in this package.
func (p *GoStdlibPackage) VariableCount() int {
	return len(p.Variables)
}

// GoStdlibFunction represents an exported function or method declaration in a stdlib package.
// It captures the full signature, parameter list, return types, and generic type parameters.
//
//nolint:tagliatelle // JSON tags match Go stdlib registry format (snake_case).
type GoStdlibFunction struct {
	Name          string             `json:"name"`
	Signature     string             `json:"signature"`
	Params        []*GoFunctionParam `json:"params"`
	Returns       []*GoReturnValue   `json:"returns"`
	IsVariadic    bool               `json:"is_variadic"`
	IsGeneric     bool               `json:"is_generic"`
	TypeParams    []*GoTypeParam     `json:"type_params"`
	ReceiverType  string             `json:"receiver_type"`
	Confidence    float32            `json:"confidence"`
	Docstring     string             `json:"docstring"`
	Deprecated    bool               `json:"deprecated"`
	DeprecatedMsg string             `json:"deprecated_msg"`
}

// GoFunctionParam represents a single parameter in a function or method signature.
//
//nolint:tagliatelle // JSON tags match Go stdlib registry format (snake_case).
type GoFunctionParam struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	IsVariadic bool   `json:"is_variadic"`
}

// GoReturnValue represents a single return value in a function or method signature.
// Named returns are common in the Go stdlib (e.g., func Read() (n int, err error)).
type GoReturnValue struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

// GoTypeParam represents a generic type parameter introduced in Go 1.18.
// Example: T in func Max[T constraints.Ordered](a, b T) T.
type GoTypeParam struct {
	Name       string `json:"name"`
	Constraint string `json:"constraint"`
}

// GoStdlibType represents an exported type declaration (struct, interface, or alias).
//
//nolint:tagliatelle // JSON tags match Go stdlib registry format (snake_case).
type GoStdlibType struct {
	Name       string                       `json:"name"`
	Kind       string                       `json:"kind"`
	Methods    map[string]*GoStdlibFunction `json:"methods"`
	Fields     []*GoStructField             `json:"fields"`
	Underlying string                       `json:"underlying"`
	IsGeneric  bool                         `json:"is_generic"`
	TypeParams []*GoTypeParam               `json:"type_params"`
	Docstring  string                       `json:"docstring"`
}

// GoStructField represents a single exported field in a struct type declaration.
type GoStructField struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Tag      string `json:"tag"`
	Exported bool   `json:"exported"`
}

// GoStdlibConstant represents an exported constant declaration in a stdlib package.
// Iota-based constants (e.g., os.O_RDONLY = iota) are flagged with IsIota.
//
//nolint:tagliatelle // JSON tags match Go stdlib registry format (snake_case).
type GoStdlibConstant struct {
	Name       string  `json:"name"`
	Type       string  `json:"type"`
	Value      string  `json:"value"`
	IsIota     bool    `json:"is_iota"`
	Confidence float32 `json:"confidence"`
	Docstring  string  `json:"docstring"`
}

// GoStdlibVariable represents an exported package-level variable declaration.
// Examples: os.Stdin (*os.File), os.Args ([]string).
type GoStdlibVariable struct {
	Name       string  `json:"name"`
	Type       string  `json:"type"`
	Confidence float32 `json:"confidence"`
	Docstring  string  `json:"docstring"`
}
