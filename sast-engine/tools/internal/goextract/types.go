package goextract

// Manifest contains registry metadata and the ordered list of extracted packages.
// It is written as manifest.json in the output directory.
//
//nolint:tagliatelle // JSON tags match Go stdlib registry format (snake_case).
type Manifest struct {
	SchemaVersion    string          `json:"schema_version"`
	RegistryVersion  string          `json:"registry_version"`
	GoVersion        VersionInfo     `json:"go_version"`
	GeneratedAt      string          `json:"generated_at"`
	GeneratorVersion string          `json:"generator_version"`
	BaseURL          string          `json:"base_url"`
	Packages         []*PackageEntry `json:"packages"`
	Statistics       *RegistryStats  `json:"statistics"`
}

// VersionInfo contains the Go release version details.
//
//nolint:tagliatelle // JSON tags match Go stdlib registry format (snake_case).
type VersionInfo struct {
	Major       int    `json:"major"`
	Minor       int    `json:"minor"`
	Patch       int    `json:"patch"`
	Full        string `json:"full"`
	ReleaseDate string `json:"release_date"`
}

// PackageEntry represents a single package's metadata entry in the manifest.
//
//nolint:tagliatelle // JSON tags match Go stdlib registry format (snake_case).
type PackageEntry struct {
	ImportPath    string `json:"import_path"`
	Checksum      string `json:"checksum"`
	FileSize      int64  `json:"file_size"`
	FunctionCount int    `json:"function_count"`
	TypeCount     int    `json:"type_count"`
	ConstantCount int    `json:"constant_count"`
}

// RegistryStats contains aggregate statistics for all extracted packages.
//
//nolint:tagliatelle // JSON tags match Go stdlib registry format (snake_case).
type RegistryStats struct {
	TotalPackages        int `json:"total_packages"`
	TotalFunctions       int `json:"total_functions"`
	TotalTypes           int `json:"total_types"`
	TotalConstants       int `json:"total_constants"`
	PackagesWithGenerics int `json:"packages_with_generics"`
}

// Package represents the complete exported API surface of a single stdlib package.
// It is written as {pkg}_stdlib.json in the output directory.
//
//nolint:tagliatelle // JSON tags match Go stdlib registry format (snake_case).
type Package struct {
	ImportPath  string               `json:"import_path"`
	GoVersion   string               `json:"go_version"`
	GeneratedAt string               `json:"generated_at"`
	Functions   map[string]*Function `json:"functions"`
	Types       map[string]*Type     `json:"types"`
	Constants   map[string]*Constant `json:"constants"`
	Variables   map[string]*Variable `json:"variables"`
}

// Function represents an exported function or method declaration.
//
//nolint:tagliatelle // JSON tags match Go stdlib registry format (snake_case).
type Function struct {
	Name          string           `json:"name"`
	Signature     string           `json:"signature"`
	Params        []*FunctionParam `json:"params"`
	Returns       []*ReturnValue   `json:"returns"`
	IsVariadic    bool             `json:"is_variadic"`
	IsGeneric     bool             `json:"is_generic"`
	TypeParams    []*TypeParam     `json:"type_params"`
	ReceiverType  string           `json:"receiver_type"`
	Confidence    float32          `json:"confidence"`
	Docstring     string           `json:"docstring"`
	Deprecated    bool             `json:"deprecated"`
	DeprecatedMsg string           `json:"deprecated_msg"`
}

// FunctionParam represents a single parameter in a function signature.
//
//nolint:tagliatelle // JSON tags match Go stdlib registry format (snake_case).
type FunctionParam struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	IsVariadic bool   `json:"is_variadic"`
}

// ReturnValue represents a single return value in a function signature.
type ReturnValue struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

// TypeParam represents a generic type parameter (Go 1.18+).
type TypeParam struct {
	Name       string `json:"name"`
	Constraint string `json:"constraint"`
}

// Type represents an exported type declaration (struct, interface, or alias).
//
//nolint:tagliatelle // JSON tags match Go stdlib registry format (snake_case).
type Type struct {
	Name       string               `json:"name"`
	Kind       string               `json:"kind"`
	Methods    map[string]*Function `json:"methods"`
	Fields     []*StructField       `json:"fields"`
	Underlying string               `json:"underlying"`
	IsGeneric  bool                 `json:"is_generic"`
	TypeParams []*TypeParam         `json:"type_params"`
	Docstring  string               `json:"docstring"`
}

// StructField represents a single exported field in a struct type.
type StructField struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Tag      string `json:"tag"`
	Exported bool   `json:"exported"`
}

// Constant represents an exported constant declaration.
//
//nolint:tagliatelle // JSON tags match Go stdlib registry format (snake_case).
type Constant struct {
	Name       string  `json:"name"`
	Type       string  `json:"type"`
	Value      string  `json:"value"`
	IsIota     bool    `json:"is_iota"`
	Confidence float32 `json:"confidence"`
	Docstring  string  `json:"docstring"`
}

// Variable represents an exported package-level variable declaration.
type Variable struct {
	Name       string  `json:"name"`
	Type       string  `json:"type"`
	Confidence float32 `json:"confidence"`
	Docstring  string  `json:"docstring"`
}
