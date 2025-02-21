package model

// Package represents a Java package, grouping multiple types.
type Package struct {
	QualifiedName string   // Fully qualified package name (e.g., "com.example")
	TopLevelTypes []string // List of top-level types in this package
	FromSource    bool     // Whether at least one reference type originates from source
	Metrics       string   // Placeholder for package-level metrics
	URL           string   // Dummy URL for the package (for debugging or references)
}

// NewPackage initializes a new Package instance.
func NewPackage(qualifiedName string, topLevelTypes []string, fromSource bool, metrics string, url string) *Package {
	return &Package{
		QualifiedName: qualifiedName,
		TopLevelTypes: topLevelTypes,
		FromSource:    fromSource,
		Metrics:       metrics,
		URL:           url,
	}
}

// ✅ Implementing Only the Provided Predicates for Package

// FromSource checks if at least one reference type in this package originates from source code.
func (p *Package) GetFromSource() bool {
	return p.FromSource
}

// GetAPrimaryQlClass returns the primary CodeQL class name for this package.
func (p *Package) GetAPrimaryQlClass() string {
	return "Package"
}

// GetATopLevelType returns a top-level type in this package.
func (p *Package) GetATopLevelType() []string {
	return p.TopLevelTypes
}

// GetMetrics provides metrics-related data for the package.
func (p *Package) GetMetrics() string {
	return p.Metrics
}

// GetURL returns a dummy URL for this package.
func (p *Package) GetURL() string {
	return p.URL
}
