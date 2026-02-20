package core

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// Mock implementation of GoStdlibLoader for interface testing.
// ---------------------------------------------------------------------------

// mockGoStdlibLoader is a test double for GoStdlibLoader.
type mockGoStdlibLoader struct {
	packageSet map[string]bool
	functions  map[string]*GoStdlibFunction
	types      map[string]*GoStdlibType
	pkgCount   int
}

func (m *mockGoStdlibLoader) ValidateStdlibImport(importPath string) bool {
	return m.packageSet[importPath]
}

func (m *mockGoStdlibLoader) GetFunction(importPath, funcName string) (*GoStdlibFunction, error) {
	key := importPath + "." + funcName
	fn, exists := m.functions[key]
	if !exists {
		return nil, errors.New("function not found")
	}
	return fn, nil
}

func (m *mockGoStdlibLoader) GetType(importPath, typeName string) (*GoStdlibType, error) {
	key := importPath + "." + typeName
	typ, exists := m.types[key]
	if !exists {
		return nil, errors.New("type not found")
	}
	return typ, nil
}

func (m *mockGoStdlibLoader) PackageCount() int {
	return m.pkgCount
}

// ---------------------------------------------------------------------------
// GoStdlibRegistry tests.
// ---------------------------------------------------------------------------

func TestNewGoStdlibRegistry(t *testing.T) {
	r := NewGoStdlibRegistry()

	assert.NotNil(t, r)
	assert.NotNil(t, r.Packages)
	assert.Nil(t, r.Manifest)
	assert.Equal(t, 0, len(r.Packages))
}

func TestGoStdlibRegistry_GetPackage(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*GoStdlibRegistry)
		importPath string
		wantNil    bool
	}{
		{
			name: "existing package",
			setup: func(r *GoStdlibRegistry) {
				r.Packages["fmt"] = NewGoStdlibPackage("fmt", "1.21")
			},
			importPath: "fmt",
			wantNil:    false,
		},
		{
			name:       "missing package",
			setup:      func(_ *GoStdlibRegistry) {},
			importPath: "nonexistent",
			wantNil:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewGoStdlibRegistry()
			tt.setup(r)
			got := r.GetPackage(tt.importPath)
			if tt.wantNil {
				assert.Nil(t, got)
			} else {
				assert.NotNil(t, got)
				assert.Equal(t, tt.importPath, got.ImportPath)
			}
		})
	}
}

func TestGoStdlibRegistry_HasPackage(t *testing.T) {
	r := NewGoStdlibRegistry()

	assert.False(t, r.HasPackage("fmt"))

	r.Packages["fmt"] = NewGoStdlibPackage("fmt", "1.21")
	assert.True(t, r.HasPackage("fmt"))
	assert.False(t, r.HasPackage("os"))
}

func TestGoStdlibRegistry_GetFunction(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*GoStdlibRegistry)
		importPath string
		funcName   string
		wantNil    bool
	}{
		{
			name: "package and function exist",
			setup: func(r *GoStdlibRegistry) {
				pkg := NewGoStdlibPackage("fmt", "1.21")
				pkg.Functions["Println"] = &GoStdlibFunction{Name: "Println", Confidence: 1.0}
				r.Packages["fmt"] = pkg
			},
			importPath: "fmt",
			funcName:   "Println",
			wantNil:    false,
		},
		{
			name:       "package does not exist",
			setup:      func(_ *GoStdlibRegistry) {},
			importPath: "fmt",
			funcName:   "Println",
			wantNil:    true,
		},
		{
			name: "package exists but function does not",
			setup: func(r *GoStdlibRegistry) {
				r.Packages["fmt"] = NewGoStdlibPackage("fmt", "1.21")
			},
			importPath: "fmt",
			funcName:   "NonExistent",
			wantNil:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewGoStdlibRegistry()
			tt.setup(r)
			got := r.GetFunction(tt.importPath, tt.funcName)
			if tt.wantNil {
				assert.Nil(t, got)
			} else {
				assert.NotNil(t, got)
				assert.Equal(t, tt.funcName, got.Name)
			}
		})
	}
}

func TestGoStdlibRegistry_GetType(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*GoStdlibRegistry)
		importPath string
		typeName   string
		wantNil    bool
	}{
		{
			name: "package and type exist",
			setup: func(r *GoStdlibRegistry) {
				pkg := NewGoStdlibPackage("fmt", "1.21")
				pkg.Types["Stringer"] = &GoStdlibType{Name: "Stringer", Kind: "interface"}
				r.Packages["fmt"] = pkg
			},
			importPath: "fmt",
			typeName:   "Stringer",
			wantNil:    false,
		},
		{
			name:       "package does not exist",
			setup:      func(_ *GoStdlibRegistry) {},
			importPath: "fmt",
			typeName:   "Stringer",
			wantNil:    true,
		},
		{
			name: "package exists but type does not",
			setup: func(r *GoStdlibRegistry) {
				r.Packages["fmt"] = NewGoStdlibPackage("fmt", "1.21")
			},
			importPath: "fmt",
			typeName:   "NonExistent",
			wantNil:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewGoStdlibRegistry()
			tt.setup(r)
			got := r.GetType(tt.importPath, tt.typeName)
			if tt.wantNil {
				assert.Nil(t, got)
			} else {
				assert.NotNil(t, got)
				assert.Equal(t, tt.typeName, got.Name)
			}
		})
	}
}

func TestGoStdlibRegistry_GetConstant(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*GoStdlibRegistry)
		importPath string
		constName  string
		wantNil    bool
	}{
		{
			name: "package and constant exist",
			setup: func(r *GoStdlibRegistry) {
				pkg := NewGoStdlibPackage("os", "1.21")
				pkg.Constants["O_RDONLY"] = &GoStdlibConstant{Name: "O_RDONLY", Type: "int", Value: "0"}
				r.Packages["os"] = pkg
			},
			importPath: "os",
			constName:  "O_RDONLY",
			wantNil:    false,
		},
		{
			name:       "package does not exist",
			setup:      func(_ *GoStdlibRegistry) {},
			importPath: "os",
			constName:  "O_RDONLY",
			wantNil:    true,
		},
		{
			name: "package exists but constant does not",
			setup: func(r *GoStdlibRegistry) {
				r.Packages["os"] = NewGoStdlibPackage("os", "1.21")
			},
			importPath: "os",
			constName:  "NonExistent",
			wantNil:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewGoStdlibRegistry()
			tt.setup(r)
			got := r.GetConstant(tt.importPath, tt.constName)
			if tt.wantNil {
				assert.Nil(t, got)
			} else {
				assert.NotNil(t, got)
				assert.Equal(t, tt.constName, got.Name)
			}
		})
	}
}

func TestGoStdlibRegistry_GetVariable(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*GoStdlibRegistry)
		importPath string
		varName    string
		wantNil    bool
	}{
		{
			name: "package and variable exist",
			setup: func(r *GoStdlibRegistry) {
				pkg := NewGoStdlibPackage("os", "1.21")
				pkg.Variables["Stdin"] = &GoStdlibVariable{Name: "Stdin", Type: "*os.File"}
				r.Packages["os"] = pkg
			},
			importPath: "os",
			varName:    "Stdin",
			wantNil:    false,
		},
		{
			name:       "package does not exist",
			setup:      func(_ *GoStdlibRegistry) {},
			importPath: "os",
			varName:    "Stdin",
			wantNil:    true,
		},
		{
			name: "package exists but variable does not",
			setup: func(r *GoStdlibRegistry) {
				r.Packages["os"] = NewGoStdlibPackage("os", "1.21")
			},
			importPath: "os",
			varName:    "NonExistent",
			wantNil:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewGoStdlibRegistry()
			tt.setup(r)
			got := r.GetVariable(tt.importPath, tt.varName)
			if tt.wantNil {
				assert.Nil(t, got)
			} else {
				assert.NotNil(t, got)
				assert.Equal(t, tt.varName, got.Name)
			}
		})
	}
}

func TestGoStdlibRegistry_PackageCount(t *testing.T) {
	r := NewGoStdlibRegistry()
	assert.Equal(t, 0, r.PackageCount())

	r.Packages["fmt"] = NewGoStdlibPackage("fmt", "1.21")
	assert.Equal(t, 1, r.PackageCount())

	r.Packages["os"] = NewGoStdlibPackage("os", "1.21")
	r.Packages["net/http"] = NewGoStdlibPackage("net/http", "1.21")
	assert.Equal(t, 3, r.PackageCount())
}

// ---------------------------------------------------------------------------
// GoManifest tests.
// ---------------------------------------------------------------------------

func TestNewGoManifest(t *testing.T) {
	m := NewGoManifest()

	assert.NotNil(t, m)
	assert.NotNil(t, m.Packages)
	assert.Equal(t, 0, len(m.Packages))
	assert.Nil(t, m.Statistics)
}

func TestGoManifest_HasPackage(t *testing.T) {
	tests := []struct {
		name       string
		packages   []*GoPackageEntry
		importPath string
		want       bool
	}{
		{
			name:       "nil packages slice",
			packages:   nil,
			importPath: "fmt",
			want:       false,
		},
		{
			name:       "empty packages slice",
			packages:   []*GoPackageEntry{},
			importPath: "fmt",
			want:       false,
		},
		{
			name: "package found",
			packages: []*GoPackageEntry{
				{ImportPath: "fmt", Checksum: "sha256:abc"},
				{ImportPath: "os", Checksum: "sha256:def"},
			},
			importPath: "fmt",
			want:       true,
		},
		{
			name: "package not found",
			packages: []*GoPackageEntry{
				{ImportPath: "os", Checksum: "sha256:def"},
			},
			importPath: "fmt",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &GoManifest{Packages: tt.packages}
			assert.Equal(t, tt.want, m.HasPackage(tt.importPath))
		})
	}
}

func TestGoManifest_GetPackageEntry(t *testing.T) {
	tests := []struct {
		name       string
		packages   []*GoPackageEntry
		importPath string
		wantNil    bool
		wantCheck  string
	}{
		{
			name:       "nil packages slice",
			packages:   nil,
			importPath: "fmt",
			wantNil:    true,
		},
		{
			name:       "empty packages slice",
			packages:   []*GoPackageEntry{},
			importPath: "fmt",
			wantNil:    true,
		},
		{
			name: "entry found",
			packages: []*GoPackageEntry{
				{ImportPath: "fmt", Checksum: "sha256:abc123", FileSize: 45000},
				{ImportPath: "os", Checksum: "sha256:def456"},
			},
			importPath: "fmt",
			wantNil:    false,
			wantCheck:  "sha256:abc123",
		},
		{
			name: "entry not found",
			packages: []*GoPackageEntry{
				{ImportPath: "os", Checksum: "sha256:def456"},
			},
			importPath: "fmt",
			wantNil:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &GoManifest{Packages: tt.packages}
			got := m.GetPackageEntry(tt.importPath)
			if tt.wantNil {
				assert.Nil(t, got)
			} else {
				assert.NotNil(t, got)
				assert.Equal(t, tt.importPath, got.ImportPath)
				assert.Equal(t, tt.wantCheck, got.Checksum)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// GoStdlibPackage tests.
// ---------------------------------------------------------------------------

func TestNewGoStdlibPackage(t *testing.T) {
	pkg := NewGoStdlibPackage("net/http", "1.21")

	assert.NotNil(t, pkg)
	assert.Equal(t, "net/http", pkg.ImportPath)
	assert.Equal(t, "1.21", pkg.GoVersion)
	assert.NotNil(t, pkg.Functions)
	assert.NotNil(t, pkg.Types)
	assert.NotNil(t, pkg.Constants)
	assert.NotNil(t, pkg.Variables)
	assert.Equal(t, 0, len(pkg.Functions))
	assert.Equal(t, 0, len(pkg.Types))
	assert.Equal(t, 0, len(pkg.Constants))
	assert.Equal(t, 0, len(pkg.Variables))
}

func TestGoStdlibPackage_Counts(t *testing.T) {
	pkg := NewGoStdlibPackage("fmt", "1.21")

	// Initial state — all counts are zero.
	assert.Equal(t, 0, pkg.FunctionCount())
	assert.Equal(t, 0, pkg.TypeCount())
	assert.Equal(t, 0, pkg.ConstantCount())
	assert.Equal(t, 0, pkg.VariableCount())

	// Populate and recheck.
	pkg.Functions["Println"] = &GoStdlibFunction{Name: "Println"}
	pkg.Functions["Sprintf"] = &GoStdlibFunction{Name: "Sprintf"}
	assert.Equal(t, 2, pkg.FunctionCount())

	pkg.Types["Stringer"] = &GoStdlibType{Name: "Stringer", Kind: "interface"}
	assert.Equal(t, 1, pkg.TypeCount())

	pkg.Constants["FlagDefault"] = &GoStdlibConstant{Name: "FlagDefault"}
	assert.Equal(t, 1, pkg.ConstantCount())

	pkg.Variables["DefaultOutput"] = &GoStdlibVariable{Name: "DefaultOutput"}
	assert.Equal(t, 1, pkg.VariableCount())
}

// ---------------------------------------------------------------------------
// JSON round-trip and snake_case tag tests.
// ---------------------------------------------------------------------------

func TestGoStdlibFunction_JSONRoundTrip(t *testing.T) {
	original := &GoStdlibFunction{
		Name:         "Println",
		Signature:    "func Println(a ...any) (n int, err error)",
		IsVariadic:   true,
		IsGeneric:    false,
		ReceiverType: "",
		Confidence:   1.0,
		Docstring:    "Println formats using the default formats.",
		Deprecated:   false,
		Params: []*GoFunctionParam{
			{Name: "a", Type: "any", IsVariadic: true},
		},
		Returns: []*GoReturnValue{
			{Type: "int", Name: "n"},
			{Type: "error", Name: "err"},
		},
	}

	data, err := json.Marshal(original)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	// Verify snake_case JSON keys are present.
	jsonStr := string(data)
	assert.Contains(t, jsonStr, `"is_variadic"`)
	assert.Contains(t, jsonStr, `"is_generic"`)
	assert.Contains(t, jsonStr, `"receiver_type"`)
	assert.Contains(t, jsonStr, `"deprecated_msg"`)

	// Round-trip: unmarshal into a new struct and verify equality.
	var decoded GoStdlibFunction
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, original.Name, decoded.Name)
	assert.Equal(t, original.Signature, decoded.Signature)
	assert.Equal(t, original.IsVariadic, decoded.IsVariadic)
	assert.Equal(t, original.IsGeneric, decoded.IsGeneric)
	assert.Equal(t, original.Confidence, decoded.Confidence)
	assert.Equal(t, original.Docstring, decoded.Docstring)
	assert.Equal(t, original.Deprecated, decoded.Deprecated)
	assert.Len(t, decoded.Params, 1)
	assert.Equal(t, "a", decoded.Params[0].Name)
	assert.True(t, decoded.Params[0].IsVariadic)
	assert.Len(t, decoded.Returns, 2)
	assert.Equal(t, "int", decoded.Returns[0].Type)
	assert.Equal(t, "n", decoded.Returns[0].Name)
}

func TestGoManifest_JSONRoundTrip(t *testing.T) {
	original := &GoManifest{
		SchemaVersion:    "1.0.0",
		RegistryVersion:  "v1",
		GeneratedAt:      "2026-02-16T10:00:00Z",
		GeneratorVersion: "1.0.0",
		BaseURL:          "https://assets.codepathfinder.dev/registries",
		GoVersion: GoVersionInfo{
			Major:       1,
			Minor:       21,
			Patch:       0,
			Full:        "1.21.0",
			ReleaseDate: "2023-08-08",
		},
		Packages: []*GoPackageEntry{
			{
				ImportPath:    "fmt",
				Checksum:      "sha256:abc123",
				FileSize:      45678,
				FunctionCount: 28,
				TypeCount:     5,
				ConstantCount: 0,
			},
		},
		Statistics: &GoRegistryStats{
			TotalPackages:        156,
			TotalFunctions:       3245,
			TotalTypes:           1823,
			TotalConstants:       892,
			PackagesWithGenerics: 12,
		},
	}

	data, err := json.Marshal(original)
	assert.NoError(t, err)

	// Verify snake_case JSON keys.
	jsonStr := string(data)
	assert.Contains(t, jsonStr, `"schema_version"`)
	assert.Contains(t, jsonStr, `"registry_version"`)
	assert.Contains(t, jsonStr, `"go_version"`)
	assert.Contains(t, jsonStr, `"generated_at"`)
	assert.Contains(t, jsonStr, `"generator_version"`)
	assert.Contains(t, jsonStr, `"base_url"`)
	assert.Contains(t, jsonStr, `"release_date"`)
	assert.Contains(t, jsonStr, `"import_path"`)
	assert.Contains(t, jsonStr, `"file_size"`)
	assert.Contains(t, jsonStr, `"function_count"`)
	assert.Contains(t, jsonStr, `"type_count"`)
	assert.Contains(t, jsonStr, `"constant_count"`)
	assert.Contains(t, jsonStr, `"total_packages"`)
	assert.Contains(t, jsonStr, `"total_functions"`)
	assert.Contains(t, jsonStr, `"packages_with_generics"`)

	var decoded GoManifest
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, original.SchemaVersion, decoded.SchemaVersion)
	assert.Equal(t, original.RegistryVersion, decoded.RegistryVersion)
	assert.Equal(t, original.GoVersion.Major, decoded.GoVersion.Major)
	assert.Equal(t, original.GoVersion.Minor, decoded.GoVersion.Minor)
	assert.Equal(t, original.GoVersion.Full, decoded.GoVersion.Full)
	assert.Equal(t, original.GoVersion.ReleaseDate, decoded.GoVersion.ReleaseDate)
	assert.Len(t, decoded.Packages, 1)
	assert.Equal(t, "fmt", decoded.Packages[0].ImportPath)
	assert.Equal(t, "sha256:abc123", decoded.Packages[0].Checksum)
	assert.Equal(t, int64(45678), decoded.Packages[0].FileSize)
	assert.NotNil(t, decoded.Statistics)
	assert.Equal(t, 156, decoded.Statistics.TotalPackages)
	assert.Equal(t, 12, decoded.Statistics.PackagesWithGenerics)
}

func TestGoStdlibPackage_JSONRoundTrip(t *testing.T) {
	pkg := NewGoStdlibPackage("fmt", "1.21.0")
	pkg.GeneratedAt = "2026-02-16T10:00:00Z"
	pkg.Functions["Println"] = &GoStdlibFunction{
		Name:       "Println",
		Signature:  "func Println(a ...any) (n int, err error)",
		IsVariadic: true,
		Confidence: 1.0,
	}
	pkg.Types["Stringer"] = &GoStdlibType{
		Name: "Stringer",
		Kind: "interface",
	}
	pkg.Constants["FlagPrecision"] = &GoStdlibConstant{
		Name:       "FlagPrecision",
		Type:       "int",
		Value:      "32",
		IsIota:     true,
		Confidence: 1.0,
	}
	pkg.Variables["DefaultOutput"] = &GoStdlibVariable{
		Name:       "DefaultOutput",
		Type:       "io.Writer",
		Confidence: 1.0,
	}

	data, err := json.Marshal(pkg)
	assert.NoError(t, err)

	// Verify snake_case keys.
	jsonStr := string(data)
	assert.Contains(t, jsonStr, `"import_path"`)
	assert.Contains(t, jsonStr, `"go_version"`)
	assert.Contains(t, jsonStr, `"generated_at"`)
	assert.Contains(t, jsonStr, `"is_iota"`)

	var decoded GoStdlibPackage
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, "fmt", decoded.ImportPath)
	assert.Equal(t, "1.21.0", decoded.GoVersion)
	assert.NotNil(t, decoded.Functions["Println"])
	assert.True(t, decoded.Functions["Println"].IsVariadic)
	assert.NotNil(t, decoded.Types["Stringer"])
	assert.Equal(t, "interface", decoded.Types["Stringer"].Kind)
	assert.NotNil(t, decoded.Constants["FlagPrecision"])
	assert.True(t, decoded.Constants["FlagPrecision"].IsIota)
	assert.NotNil(t, decoded.Variables["DefaultOutput"])
}

func TestGoStdlibType_JSONRoundTrip(t *testing.T) {
	original := &GoStdlibType{
		Name:      "ResponseWriter",
		Kind:      "interface",
		IsGeneric: false,
		Docstring: "ResponseWriter is used by an HTTP handler.",
		Methods: map[string]*GoStdlibFunction{
			"WriteHeader": {Name: "WriteHeader", Signature: "WriteHeader(statusCode int)"},
		},
		Fields: []*GoStructField{
			{Name: "Code", Type: "int", Tag: `json:"code"`, Exported: true},
		},
	}

	data, err := json.Marshal(original)
	assert.NoError(t, err)

	jsonStr := string(data)
	assert.Contains(t, jsonStr, `"is_generic"`)
	assert.Contains(t, jsonStr, `"type_params"`)

	var decoded GoStdlibType
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, original.Name, decoded.Name)
	assert.Equal(t, original.Kind, decoded.Kind)
	assert.Equal(t, original.IsGeneric, decoded.IsGeneric)
	assert.NotNil(t, decoded.Methods["WriteHeader"])
	assert.Len(t, decoded.Fields, 1)
	assert.True(t, decoded.Fields[0].Exported)
}

func TestGoFunctionParam_JSONSnakeCase(t *testing.T) {
	param := GoFunctionParam{Name: "values", Type: "[]int", IsVariadic: true}

	data, err := json.Marshal(param)
	assert.NoError(t, err)

	jsonStr := string(data)
	assert.Contains(t, jsonStr, `"is_variadic":true`)
	assert.NotContains(t, jsonStr, `"isVariadic"`)
}

func TestGoTypeParam_JSONRoundTrip(t *testing.T) {
	original := GoTypeParam{Name: "T", Constraint: "constraints.Ordered"}

	data, err := json.Marshal(original)
	assert.NoError(t, err)

	var decoded GoTypeParam
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, original.Name, decoded.Name)
	assert.Equal(t, original.Constraint, decoded.Constraint)
}

func TestGoReturnValue_JSONRoundTrip(t *testing.T) {
	original := GoReturnValue{Type: "[]byte", Name: "data"}

	data, err := json.Marshal(original)
	assert.NoError(t, err)

	var decoded GoReturnValue
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, original.Type, decoded.Type)
	assert.Equal(t, original.Name, decoded.Name)
}

func TestGoStdlibConstant_JSONSnakeCase(t *testing.T) {
	c := GoStdlibConstant{
		Name:       "O_RDONLY",
		Type:       "int",
		Value:      "0",
		IsIota:     true,
		Confidence: 1.0,
		Docstring:  "O_RDONLY is the flag for read-only access.",
	}

	data, err := json.Marshal(c)
	assert.NoError(t, err)

	jsonStr := string(data)
	assert.Contains(t, jsonStr, `"is_iota":true`)
	assert.NotContains(t, jsonStr, `"isIota"`)
}

func TestGoStdlibVariable_JSONRoundTrip(t *testing.T) {
	original := GoStdlibVariable{
		Name:       "Stdin",
		Type:       "*os.File",
		Confidence: 1.0,
		Docstring:  "Stdin is the standard input.",
	}

	data, err := json.Marshal(original)
	assert.NoError(t, err)

	var decoded GoStdlibVariable
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, original.Name, decoded.Name)
	assert.Equal(t, original.Type, decoded.Type)
	assert.Equal(t, original.Confidence, decoded.Confidence)
}

func TestGoStructField_JSONRoundTrip(t *testing.T) {
	original := GoStructField{
		Name:     "StatusCode",
		Type:     "int",
		Tag:      `json:"status_code"`,
		Exported: true,
	}

	data, err := json.Marshal(original)
	assert.NoError(t, err)

	var decoded GoStructField
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, original.Name, decoded.Name)
	assert.Equal(t, original.Type, decoded.Type)
	assert.Equal(t, original.Tag, decoded.Tag)
	assert.Equal(t, original.Exported, decoded.Exported)
}

func TestGoRegistryStats_JSONSnakeCase(t *testing.T) {
	stats := GoRegistryStats{
		TotalPackages:        156,
		TotalFunctions:       3245,
		TotalTypes:           1823,
		TotalConstants:       892,
		PackagesWithGenerics: 12,
	}

	data, err := json.Marshal(stats)
	assert.NoError(t, err)

	jsonStr := string(data)
	assert.Contains(t, jsonStr, `"total_packages"`)
	assert.Contains(t, jsonStr, `"total_functions"`)
	assert.Contains(t, jsonStr, `"total_types"`)
	assert.Contains(t, jsonStr, `"total_constants"`)
	assert.Contains(t, jsonStr, `"packages_with_generics"`)

	var decoded GoRegistryStats
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, stats.TotalPackages, decoded.TotalPackages)
	assert.Equal(t, stats.PackagesWithGenerics, decoded.PackagesWithGenerics)
}

func TestGoPackageEntry_JSONSnakeCase(t *testing.T) {
	entry := GoPackageEntry{
		ImportPath:    "net/http",
		Checksum:      "sha256:abc123",
		FileSize:      123456,
		FunctionCount: 87,
		TypeCount:     42,
		ConstantCount: 35,
	}

	data, err := json.Marshal(entry)
	assert.NoError(t, err)

	jsonStr := string(data)
	assert.Contains(t, jsonStr, `"import_path"`)
	assert.Contains(t, jsonStr, `"file_size"`)
	assert.Contains(t, jsonStr, `"function_count"`)
	assert.Contains(t, jsonStr, `"type_count"`)
	assert.Contains(t, jsonStr, `"constant_count"`)

	var decoded GoPackageEntry
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, entry.ImportPath, decoded.ImportPath)
	assert.Equal(t, entry.Checksum, decoded.Checksum)
	assert.Equal(t, entry.FileSize, decoded.FileSize)
	assert.Equal(t, entry.FunctionCount, decoded.FunctionCount)
}

// ---------------------------------------------------------------------------
// GoCallEdge tests.
// ---------------------------------------------------------------------------

func TestNewGoCallEdge(t *testing.T) {
	edge := NewGoCallEdge("main.main", "fmt.Println")

	assert.NotNil(t, edge)
	assert.Equal(t, "main.main", edge.Source)
	assert.Equal(t, "fmt.Println", edge.Target)
	assert.Equal(t, "call", edge.CallType)
	assert.Equal(t, uint32(0), edge.LineNumber)
	assert.Nil(t, edge.Arguments)
	assert.Empty(t, edge.FilePath)
	assert.False(t, edge.IsExternal)
	assert.False(t, edge.IsStdlib)
	assert.Equal(t, float32(0), edge.Confidence)
}

func TestGoCallEdge_FieldAssignment(t *testing.T) {
	edge := NewGoCallEdge("myapp.handler.HandleRequest", "fmt.Fprintf")
	edge.CallType = "stdlib_call"
	edge.LineNumber = 42
	edge.Arguments = []string{"w", `"hello %s"`, "name"}
	edge.FilePath = "/project/handler.go"
	edge.IsExternal = true
	edge.IsStdlib = true
	edge.Confidence = 1.0

	assert.Equal(t, "myapp.handler.HandleRequest", edge.Source)
	assert.Equal(t, "fmt.Fprintf", edge.Target)
	assert.Equal(t, "stdlib_call", edge.CallType)
	assert.Equal(t, uint32(42), edge.LineNumber)
	assert.Len(t, edge.Arguments, 3)
	assert.Equal(t, "/project/handler.go", edge.FilePath)
	assert.True(t, edge.IsExternal)
	assert.True(t, edge.IsStdlib)
	assert.Equal(t, float32(1.0), edge.Confidence)
}

// ---------------------------------------------------------------------------
// GoStdlibLoader interface tests (via mock).
// ---------------------------------------------------------------------------

func TestGoStdlibLoader_ValidateStdlibImport(t *testing.T) {
	loader := &mockGoStdlibLoader{
		packageSet: map[string]bool{"fmt": true, "os": true},
		pkgCount:   2,
	}

	assert.True(t, loader.ValidateStdlibImport("fmt"))
	assert.True(t, loader.ValidateStdlibImport("os"))
	assert.False(t, loader.ValidateStdlibImport("github.com/example/myapp"))
	assert.False(t, loader.ValidateStdlibImport(""))
}

func TestGoStdlibLoader_GetFunction(t *testing.T) {
	fn := &GoStdlibFunction{Name: "Println", Confidence: 1.0}
	loader := &mockGoStdlibLoader{
		functions: map[string]*GoStdlibFunction{
			"fmt.Println": fn,
		},
	}

	// Found.
	got, err := loader.GetFunction("fmt", "Println")
	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, "Println", got.Name)

	// Not found.
	got, err = loader.GetFunction("fmt", "NonExistent")
	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestGoStdlibLoader_GetType(t *testing.T) {
	typ := &GoStdlibType{Name: "Stringer", Kind: "interface"}
	loader := &mockGoStdlibLoader{
		types: map[string]*GoStdlibType{
			"fmt.Stringer": typ,
		},
	}

	// Found.
	got, err := loader.GetType("fmt", "Stringer")
	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, "Stringer", got.Name)

	// Not found.
	got, err = loader.GetType("fmt", "NonExistent")
	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestGoStdlibLoader_PackageCount(t *testing.T) {
	loader := &mockGoStdlibLoader{pkgCount: 156}
	assert.Equal(t, 156, loader.PackageCount())
}

// ---------------------------------------------------------------------------
// GoModuleRegistry.StdlibLoader integration test.
// ---------------------------------------------------------------------------

func TestGoModuleRegistry_StdlibLoader(t *testing.T) {
	reg := NewGoModuleRegistry()

	// Initially nil.
	assert.Nil(t, reg.StdlibLoader)

	// Attach a mock loader.
	loader := &mockGoStdlibLoader{
		packageSet: map[string]bool{"fmt": true},
		functions: map[string]*GoStdlibFunction{
			"fmt.Println": {Name: "Println", Confidence: 1.0},
		},
		pkgCount: 1,
	}
	reg.StdlibLoader = loader

	assert.NotNil(t, reg.StdlibLoader)
	assert.True(t, reg.StdlibLoader.ValidateStdlibImport("fmt"))
	assert.False(t, reg.StdlibLoader.ValidateStdlibImport("os"))
	assert.Equal(t, 1, reg.StdlibLoader.PackageCount())

	fn, err := reg.StdlibLoader.GetFunction("fmt", "Println")
	assert.NoError(t, err)
	assert.NotNil(t, fn)
	assert.Equal(t, float32(1.0), fn.Confidence)
}

// ---------------------------------------------------------------------------
// Integration test: full stdlib registry workflow.
// ---------------------------------------------------------------------------

func TestGoStdlibRegistry_Integration(t *testing.T) {
	r := NewGoStdlibRegistry()

	// Build a manifest.
	m := NewGoManifest()
	m.SchemaVersion = "1.0.0"
	m.GoVersion = GoVersionInfo{Major: 1, Minor: 21, Full: "1.21.0", ReleaseDate: "2023-08-08"}
	m.Statistics = &GoRegistryStats{TotalPackages: 2, TotalFunctions: 30}
	m.Packages = append(m.Packages, &GoPackageEntry{
		ImportPath:    "fmt",
		Checksum:      "sha256:abc",
		FunctionCount: 28,
	})
	m.Packages = append(m.Packages, &GoPackageEntry{
		ImportPath:   "os",
		Checksum:     "sha256:def",
		ConstantCount: 5,
	})
	r.Manifest = m

	// Build fmt package.
	fmtPkg := NewGoStdlibPackage("fmt", "1.21.0")
	fmtPkg.Functions["Println"] = &GoStdlibFunction{
		Name:       "Println",
		Signature:  "func Println(a ...any) (n int, err error)",
		IsVariadic: true,
		Confidence: 1.0,
		Params:     []*GoFunctionParam{{Name: "a", Type: "any", IsVariadic: true}},
		Returns:    []*GoReturnValue{{Type: "int", Name: "n"}, {Type: "error", Name: "err"}},
	}
	fmtPkg.Functions["Sprintf"] = &GoStdlibFunction{
		Name:       "Sprintf",
		Signature:  "func Sprintf(format string, a ...any) string",
		IsVariadic: true,
		Confidence: 1.0,
		Returns:    []*GoReturnValue{{Type: "string"}},
	}
	fmtPkg.Types["Stringer"] = &GoStdlibType{
		Name: "Stringer",
		Kind: "interface",
		Methods: map[string]*GoStdlibFunction{
			"String": {Name: "String", Returns: []*GoReturnValue{{Type: "string"}}},
		},
	}
	r.Packages["fmt"] = fmtPkg

	// Build os package.
	osPkg := NewGoStdlibPackage("os", "1.21.0")
	osPkg.Constants["O_RDONLY"] = &GoStdlibConstant{Name: "O_RDONLY", Type: "int", Value: "0", IsIota: true, Confidence: 1.0}
	osPkg.Variables["Stdin"] = &GoStdlibVariable{Name: "Stdin", Type: "*os.File", Confidence: 1.0}
	r.Packages["os"] = osPkg

	// Assertions on the registry.
	assert.Equal(t, 2, r.PackageCount())
	assert.True(t, r.HasPackage("fmt"))
	assert.True(t, r.HasPackage("os"))
	assert.False(t, r.HasPackage("net/http"))

	// fmt function lookups.
	printlnFn := r.GetFunction("fmt", "Println")
	assert.NotNil(t, printlnFn)
	assert.True(t, printlnFn.IsVariadic)
	assert.Len(t, printlnFn.Returns, 2)

	sprintf := r.GetFunction("fmt", "Sprintf")
	assert.NotNil(t, sprintf)
	assert.Equal(t, "string", sprintf.Returns[0].Type)

	// fmt type lookup.
	stringer := r.GetType("fmt", "Stringer")
	assert.NotNil(t, stringer)
	assert.Equal(t, "interface", stringer.Kind)
	assert.NotNil(t, stringer.Methods["String"])

	// os constant lookup.
	oRdonly := r.GetConstant("os", "O_RDONLY")
	assert.NotNil(t, oRdonly)
	assert.Equal(t, "0", oRdonly.Value)
	assert.True(t, oRdonly.IsIota)

	// os variable lookup.
	stdin := r.GetVariable("os", "Stdin")
	assert.NotNil(t, stdin)
	assert.Equal(t, "*os.File", stdin.Type)

	// Missing lookups.
	assert.Nil(t, r.GetFunction("nonexistent", "Foo"))
	assert.Nil(t, r.GetType("nonexistent", "Bar"))
	assert.Nil(t, r.GetConstant("nonexistent", "BAZ"))
	assert.Nil(t, r.GetVariable("nonexistent", "Qux"))

	// Manifest queries.
	assert.True(t, m.HasPackage("fmt"))
	assert.True(t, m.HasPackage("os"))
	assert.False(t, m.HasPackage("net/http"))

	fmtEntry := m.GetPackageEntry("fmt")
	assert.NotNil(t, fmtEntry)
	assert.Equal(t, "sha256:abc", fmtEntry.Checksum)
	assert.Equal(t, 28, fmtEntry.FunctionCount)

	assert.Nil(t, m.GetPackageEntry("net/http"))

	// Package counts.
	assert.Equal(t, 2, fmtPkg.FunctionCount())
	assert.Equal(t, 1, fmtPkg.TypeCount())
	assert.Equal(t, 0, fmtPkg.ConstantCount())
	assert.Equal(t, 0, fmtPkg.VariableCount())

	assert.Equal(t, 0, osPkg.FunctionCount())
	assert.Equal(t, 1, osPkg.ConstantCount())
	assert.Equal(t, 1, osPkg.VariableCount())
}
