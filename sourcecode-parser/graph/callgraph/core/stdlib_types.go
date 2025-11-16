package core

// StdlibRegistry holds all Python stdlib module registries.
type StdlibRegistry struct {
	Modules  map[string]*StdlibModule
	Manifest *Manifest
}

// Manifest contains metadata about the stdlib registry.
//
//nolint:tagliatelle // JSON tags match Python-generated registry format (snake_case).
type Manifest struct {
	SchemaVersion    string            `json:"schema_version"`
	RegistryVersion  string            `json:"registry_version"`
	PythonVersion    PythonVersionInfo `json:"python_version"`
	GeneratedAt      string            `json:"generated_at"`
	GeneratorVersion string            `json:"generator_version"`
	BaseURL          string            `json:"base_url"`
	Modules          []*ModuleEntry    `json:"modules"`
	Statistics       *RegistryStats    `json:"statistics"`
}

// PythonVersionInfo contains Python version details.
type PythonVersionInfo struct {
	Major int    `json:"major"`
	Minor int    `json:"minor"`
	Patch int    `json:"patch"`
	Full  string `json:"full"`
}

// ModuleEntry represents a single module in the manifest.
//
//nolint:tagliatelle // JSON tags match Python-generated registry format (snake_case).
type ModuleEntry struct {
	Name      string `json:"name"`
	File      string `json:"file"`
	URL       string `json:"url"`
	SizeBytes int64  `json:"size_bytes"`
	Checksum  string `json:"checksum"`
}

// RegistryStats contains aggregate statistics.
//
//nolint:tagliatelle // JSON tags match Python-generated registry format (snake_case).
type RegistryStats struct {
	TotalModules    int `json:"total_modules"`
	TotalFunctions  int `json:"total_functions"`
	TotalClasses    int `json:"total_classes"`
	TotalConstants  int `json:"total_constants"`
	TotalAttributes int `json:"total_attributes"`
}

// StdlibModule represents a single stdlib module registry.
//
//nolint:tagliatelle // JSON tags match Python-generated registry format (snake_case).
type StdlibModule struct {
	Module        string                      `json:"module"`
	PythonVersion string                      `json:"python_version"`
	GeneratedAt   string                      `json:"generated_at"`
	Functions     map[string]*StdlibFunction  `json:"functions"`
	Classes       map[string]*StdlibClass     `json:"classes"`
	Constants     map[string]*StdlibConstant  `json:"constants"`
	Attributes    map[string]*StdlibAttribute `json:"attributes"`
}

// StdlibFunction represents a function in a stdlib module.
//
//nolint:tagliatelle // JSON tags match Python-generated registry format (snake_case).
type StdlibFunction struct {
	ReturnType string           `json:"return_type"`
	Confidence float32          `json:"confidence"`
	Params     []*FunctionParam `json:"params"`
	Source     string           `json:"source"`
	Docstring  string           `json:"docstring,omitempty"`
}

// FunctionParam represents a function parameter.
type FunctionParam struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
}

// StdlibClass represents a class in a stdlib module.
type StdlibClass struct {
	Type      string                     `json:"type"`
	Methods   map[string]*StdlibFunction `json:"methods"`
	Docstring string                     `json:"docstring,omitempty"`
}

// StdlibConstant represents a module-level constant.
//
//nolint:tagliatelle // JSON tags match Python-generated registry format (snake_case).
type StdlibConstant struct {
	Type             string  `json:"type"`
	Value            string  `json:"value"`
	Confidence       float32 `json:"confidence"`
	PlatformSpecific bool    `json:"platform_specific,omitempty"`
}

// StdlibAttribute represents a module-level attribute (os.environ, sys.modules, etc.).
//
//nolint:tagliatelle // JSON tags match Python-generated registry format (snake_case).
type StdlibAttribute struct {
	Type        string  `json:"type"`
	BehavesLike string  `json:"behaves_like,omitempty"`
	Confidence  float32 `json:"confidence"`
	Docstring   string  `json:"docstring,omitempty"`
}

// NewStdlibRegistry creates a new stdlib registry.
func NewStdlibRegistry() *StdlibRegistry {
	return &StdlibRegistry{
		Modules: make(map[string]*StdlibModule),
	}
}

// GetModule returns the registry for a specific module.
func (r *StdlibRegistry) GetModule(moduleName string) *StdlibModule {
	return r.Modules[moduleName]
}

// HasModule checks if a module exists in the registry.
func (r *StdlibRegistry) HasModule(moduleName string) bool {
	_, exists := r.Modules[moduleName]
	return exists
}

// GetFunction returns a function from a module.
func (r *StdlibRegistry) GetFunction(moduleName, functionName string) *StdlibFunction {
	module := r.GetModule(moduleName)
	if module == nil {
		return nil
	}
	return module.Functions[functionName]
}

// GetClass returns a class from a module.
func (r *StdlibRegistry) GetClass(moduleName, className string) *StdlibClass {
	module := r.GetModule(moduleName)
	if module == nil {
		return nil
	}
	return module.Classes[className]
}

// GetConstant returns a constant from a module.
func (r *StdlibRegistry) GetConstant(moduleName, constantName string) *StdlibConstant {
	module := r.GetModule(moduleName)
	if module == nil {
		return nil
	}
	return module.Constants[constantName]
}

// GetAttribute returns an attribute from a module.
func (r *StdlibRegistry) GetAttribute(moduleName, attributeName string) *StdlibAttribute {
	module := r.GetModule(moduleName)
	if module == nil {
		return nil
	}
	return module.Attributes[attributeName]
}

// ModuleCount returns the number of loaded modules.
func (r *StdlibRegistry) ModuleCount() int {
	return len(r.Modules)
}
