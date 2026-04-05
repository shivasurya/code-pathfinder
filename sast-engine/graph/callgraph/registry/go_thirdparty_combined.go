package registry

import (
	"fmt"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// GoThirdPartyCombinedLoader wraps a CDN-based loader and a local loader into a single
// GoThirdPartyLoader. The CDN is checked first (fast, pre-validated metadata) and the
// local loader is used as a fallback (slower, but covers all project dependencies).
//
// CDN-first resolution strategy:
//   - If the CDN declares the package (ValidateImport → true):
//     — Successful lookup (non-nil result, nil error) → return CDN result.
//     — Authoritative miss (nil result, nil error) → the CDN has the package but the
//     symbol doesn't exist in it; do NOT fall through to local (CDN data is
//     considered authoritative for its packages, so falling back would mask
//     extraction bugs).
//     — Transient failure (non-nil error) → network / parse / download issue;
//     fall through to local so a momentary CDN outage doesn't break analysis.
//   - If the CDN does not know the package (ValidateImport → false):
//     — Fall through to local unconditionally.
//
// Either loader may be nil. Nil loaders are silently skipped.
// If both loaders are nil, all lookups return descriptive errors.
type GoThirdPartyCombinedLoader struct {
	cdnLoader   core.GoThirdPartyLoader
	localLoader core.GoThirdPartyLoader
}

// NewGoThirdPartyCombinedLoader creates a GoThirdPartyCombinedLoader from a CDN and a
// local loader. Either argument may be nil; the loader degrades gracefully.
func NewGoThirdPartyCombinedLoader(
	cdnLoader core.GoThirdPartyLoader,
	localLoader core.GoThirdPartyLoader,
) *GoThirdPartyCombinedLoader {
	return &GoThirdPartyCombinedLoader{
		cdnLoader:   cdnLoader,
		localLoader: localLoader,
	}
}

// ValidateImport returns true if either the CDN or the local loader recognises the
// given import path. Both loaders are consulted (OR logic).
func (c *GoThirdPartyCombinedLoader) ValidateImport(importPath string) bool {
	if c.cdnLoader != nil && c.cdnLoader.ValidateImport(importPath) {
		return true
	}
	if c.localLoader != nil && c.localLoader.ValidateImport(importPath) {
		return true
	}
	return false
}

// GetType retrieves the metadata for a named type in the given third-party package.
// The CDN is checked first; the local loader is used as a fallback according to the
// CDN-first resolution strategy described on GoThirdPartyCombinedLoader.
func (c *GoThirdPartyCombinedLoader) GetType(importPath, typeName string) (*core.GoStdlibType, error) {
	if c.cdnLoader != nil && c.cdnLoader.ValidateImport(importPath) {
		typ, err := c.cdnLoader.GetType(importPath, typeName)
		switch {
		case err != nil:
			// Transient failure (network, parse, download) — fall through to local.
		case typ != nil:
			return typ, nil
		default:
			// Authoritative miss: CDN has the package but not this type.
			// Do not fall back — CDN metadata is considered authoritative for its packages.
			return nil, fmt.Errorf("type %s.%s not found in CDN (authoritative)", importPath, typeName)
		}
	}

	if c.localLoader != nil {
		return c.localLoader.GetType(importPath, typeName)
	}

	return nil, fmt.Errorf("type %s.%s not found in any loader", importPath, typeName)
}

// GetFunction retrieves the metadata for a named package-level function in the given
// third-party package. Uses the same CDN-first resolution strategy as GetType.
func (c *GoThirdPartyCombinedLoader) GetFunction(importPath, funcName string) (*core.GoStdlibFunction, error) {
	if c.cdnLoader != nil && c.cdnLoader.ValidateImport(importPath) {
		fn, err := c.cdnLoader.GetFunction(importPath, funcName)
		switch {
		case err != nil:
			// Transient failure (network, parse, download) — fall through to local.
		case fn != nil:
			return fn, nil
		default:
			// Authoritative miss: CDN has the package but not this function.
			// Do not fall back — CDN metadata is considered authoritative for its packages.
			return nil, fmt.Errorf("function %s.%s not found in CDN (authoritative)", importPath, funcName)
		}
	}

	if c.localLoader != nil {
		return c.localLoader.GetFunction(importPath, funcName)
	}

	return nil, fmt.Errorf("function %s.%s not found in any loader", importPath, funcName)
}

// PackageCount returns the sum of packages from both loaders.
// Packages present in both CDN and local are counted once per loader; slight
// over-counting is acceptable since CDN covers popular packages while local covers
// project-specific dependencies — overlap is minimal in practice.
func (c *GoThirdPartyCombinedLoader) PackageCount() int {
	cdnCount := 0
	localCount := 0
	if c.cdnLoader != nil {
		cdnCount = c.cdnLoader.PackageCount()
	}
	if c.localLoader != nil {
		localCount = c.localLoader.PackageCount()
	}
	return cdnCount + localCount
}
