package registry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"

	sitter "github.com/smacker/go-tree-sitter"
	golang "github.com/smacker/go-tree-sitter/golang"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/output"
)

// cacheIndexVersion is the schema version written into cache-index.json.
// Increment when the cache format changes to force re-extraction on upgrade.
const cacheIndexVersion = "1.0.0"

// errPackageSourceNotFound is returned by getOrLoadPackage when no source
// directory can be located for the requested import path.
var errPackageSourceNotFound = errors.New("go-thirdparty: package source not found in vendor/ or GOMODCACHE")

// cacheIndexEntry is one record in cache-index.json.
type cacheIndexEntry struct {
	Version  string    `json:"version"`
	File     string    `json:"file"`
	CachedAt time.Time `json:"cachedAt"`
}

// cacheIndex is the in-memory representation of cache-index.json.
type cacheIndex struct {
	Version string                      `json:"version"`
	Entries map[string]*cacheIndexEntry `json:"entries"`
}

// GoThirdPartyLocalLoader extracts type metadata from third-party Go packages
// found in vendor/ or GOMODCACHE. Uses tree-sitter for lightweight parsing.
// Implements core.GoThirdPartyLoader.
type GoThirdPartyLocalLoader struct {
	projectRoot    string
	moduleVersions map[string]string                // import path → version (from go.mod require)
	packageCache   map[string]*core.GoStdlibPackage // import path → extracted package (in-memory)
	cacheMutex     sync.RWMutex
	cacheDir       string      // disk cache directory: {userCacheDir}/code-pathfinder/go-thirdparty/{projectHash}/
	diskIndex      *cacheIndex // loaded from cache-index.json; nil when disk cache is unavailable
	logger         *output.Logger
}

// NewGoThirdPartyLocalLoader creates a loader that finds and parses third-party
// Go packages from vendor/ or GOMODCACHE.
//
// When refreshCache is true (set by --refresh-rules on the CLI), the existing
// go-thirdparty disk cache for this project is deleted and rebuilt from source.
func NewGoThirdPartyLocalLoader(projectRoot string, refreshCache bool, logger *output.Logger) *GoThirdPartyLocalLoader {
	loader := &GoThirdPartyLocalLoader{
		projectRoot:  projectRoot,
		packageCache: make(map[string]*core.GoStdlibPackage),
		logger:       logger,
	}
	loader.moduleVersions = parseGoModRequires(projectRoot)
	if logger != nil {
		logger.Debug("Go third-party local loader: found %d dependencies in go.mod", len(loader.moduleVersions))
	}
	loader.cacheDir = goThirdPartyCacheDir(projectRoot)
	loader.initDiskCache(refreshCache)
	return loader
}

// goThirdPartyCacheDir returns the project-specific disk cache directory.
// Path: {os.UserCacheDir}/code-pathfinder/go-thirdparty/{sha256(projectRoot)[:12]}.
func goThirdPartyCacheDir(projectRoot string) string {
	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}
	h := sha256.Sum256([]byte(projectRoot))
	projectHash := hex.EncodeToString(h[:])[:12]
	return filepath.Join(base, "code-pathfinder", "go-thirdparty", projectHash)
}

// initDiskCache prepares the on-disk cache directory and loads cache-index.json.
// If refreshCache is true, the directory is wiped before loading (always a miss).
func (l *GoThirdPartyLocalLoader) initDiskCache(refreshCache bool) {
	if refreshCache {
		if err := os.RemoveAll(l.cacheDir); err != nil && l.logger != nil {
			l.logger.Debug("go-thirdparty: failed to flush cache dir %s: %v", l.cacheDir, err)
		}
	}
	if err := os.MkdirAll(l.cacheDir, 0o755); err != nil {
		if l.logger != nil {
			l.logger.Debug("go-thirdparty: could not create cache dir %s: %v", l.cacheDir, err)
		}
		return
	}
	l.diskIndex = l.loadCacheIndex()
}

// loadCacheIndex reads cache-index.json from disk. Returns an empty index on any error.
func (l *GoThirdPartyLocalLoader) loadCacheIndex() *cacheIndex {
	indexPath := filepath.Join(l.cacheDir, "cache-index.json")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return &cacheIndex{Version: cacheIndexVersion, Entries: make(map[string]*cacheIndexEntry)}
	}
	var idx cacheIndex
	if err := json.Unmarshal(data, &idx); err != nil || idx.Entries == nil {
		return &cacheIndex{Version: cacheIndexVersion, Entries: make(map[string]*cacheIndexEntry)}
	}
	return &idx
}

// saveCacheIndex writes the current diskIndex to cache-index.json.
// Called while the write lock is held.
func (l *GoThirdPartyLocalLoader) saveCacheIndex() {
	if l.diskIndex == nil || l.cacheDir == "" {
		return
	}
	data, err := json.MarshalIndent(l.diskIndex, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(l.cacheDir, "cache-index.json"), data, 0o644)
}

// encodeCachePath converts an import path to a safe filename component.
// e.g. "gorm.io/gorm" → "gorm.io_gorm".
func encodeCachePath(importPath string) string {
	return strings.ReplaceAll(importPath, "/", "_")
}

// ValidateImport reports whether the import path is a known third-party dependency.
func (l *GoThirdPartyLocalLoader) ValidateImport(importPath string) bool {
	// Check if any known module is a prefix of the import path.
	// e.g., importPath "gorm.io/gorm" matches module "gorm.io/gorm"
	// e.g., importPath "github.com/gin-gonic/gin/binding" matches module "github.com/gin-gonic/gin"
	for modPath := range l.moduleVersions {
		if importPath == modPath || strings.HasPrefix(importPath, modPath+"/") {
			return true
		}
	}
	return false
}

// GetFunction returns function metadata for a third-party package function.
func (l *GoThirdPartyLocalLoader) GetFunction(importPath, funcName string) (*core.GoStdlibFunction, error) {
	pkg, err := l.getOrLoadPackage(importPath)
	if err != nil || pkg == nil {
		return nil, err
	}
	fn, ok := pkg.Functions[funcName]
	if !ok {
		return nil, fmt.Errorf("function %s not found in %s", funcName, importPath)
	}
	return fn, nil
}

// GetType returns type metadata for a third-party package type.
func (l *GoThirdPartyLocalLoader) GetType(importPath, typeName string) (*core.GoStdlibType, error) {
	pkg, err := l.getOrLoadPackage(importPath)
	if err != nil || pkg == nil {
		return nil, err
	}
	typ, ok := pkg.Types[typeName]
	if !ok {
		return nil, fmt.Errorf("type %s not found in %s", typeName, importPath)
	}
	return typ, nil
}

// PackageCount returns the number of known third-party dependencies.
func (l *GoThirdPartyLocalLoader) PackageCount() int {
	return len(l.moduleVersions)
}

// getOrLoadPackage retrieves a package from the in-memory cache, the disk cache,
// or by parsing from vendor/GOMODCACHE (in that priority order).
func (l *GoThirdPartyLocalLoader) getOrLoadPackage(importPath string) (*core.GoStdlibPackage, error) {
	// Fast path: in-memory cache (includes negative results stored as nil).
	l.cacheMutex.RLock()
	if pkg, ok := l.packageCache[importPath]; ok {
		l.cacheMutex.RUnlock()
		return pkg, nil
	}
	l.cacheMutex.RUnlock()

	// Slow path: disk cache then source parse.
	l.cacheMutex.Lock()
	defer l.cacheMutex.Unlock()

	// Double-check under write lock.
	if pkg, ok := l.packageCache[importPath]; ok {
		return pkg, nil
	}

	// Disk cache hit: version must match go.mod require version.
	if pkg := l.loadFromDiskCache(importPath); pkg != nil {
		l.packageCache[importPath] = pkg
		return pkg, nil
	}

	// Parse from vendor/ or GOMODCACHE.
	srcDir := l.findPackageSource(importPath)
	if srcDir == "" {
		l.packageCache[importPath] = nil
		return nil, errPackageSourceNotFound
	}

	pkg, err := extractGoPackageWithTreeSitter(importPath, srcDir)
	if err != nil {
		if l.logger != nil {
			l.logger.Debug("Failed to extract third-party package %s: %v", importPath, err)
		}
		l.packageCache[importPath] = nil
		return nil, err
	}

	l.packageCache[importPath] = pkg
	if l.logger != nil {
		l.logger.Debug("Extracted third-party package %s: %d types, %d functions",
			importPath, len(pkg.Types), len(pkg.Functions))
	}

	// Persist to disk cache for subsequent runs.
	l.writeToDiskCache(importPath, pkg)
	return pkg, nil
}

// loadFromDiskCache attempts to read a GoStdlibPackage from the disk cache.
// Returns nil on any cache miss, version mismatch, or read error.
func (l *GoThirdPartyLocalLoader) loadFromDiskCache(importPath string) *core.GoStdlibPackage {
	if l.diskIndex == nil || l.cacheDir == "" {
		return nil
	}
	entry, ok := l.diskIndex.Entries[importPath]
	if !ok {
		return nil
	}
	// Version mismatch → stale cache, re-extract.
	if wantVer := l.moduleVersions[l.moduleKeyFor(importPath)]; wantVer != "" && entry.Version != wantVer {
		return nil
	}
	data, err := os.ReadFile(filepath.Join(l.cacheDir, entry.File))
	if err != nil {
		return nil
	}
	var pkg core.GoStdlibPackage
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil
	}
	if l.logger != nil {
		l.logger.Debug("go-thirdparty: disk cache hit for %s (%s)", importPath, entry.Version)
	}
	return &pkg
}

// writeToDiskCache serialises pkg to a JSON file and updates cache-index.json.
// Errors are logged at debug level and silently ignored (cache is best-effort).
func (l *GoThirdPartyLocalLoader) writeToDiskCache(importPath string, pkg *core.GoStdlibPackage) {
	if l.diskIndex == nil || l.cacheDir == "" {
		return
	}
	version := l.moduleVersions[l.moduleKeyFor(importPath)]
	fileName := encodeCachePath(importPath) + "@" + version + ".json"

	data, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		return
	}
	if err := os.WriteFile(filepath.Join(l.cacheDir, fileName), data, 0o644); err != nil {
		if l.logger != nil {
			l.logger.Debug("go-thirdparty: failed to write disk cache for %s: %v", importPath, err)
		}
		return
	}
	l.diskIndex.Entries[importPath] = &cacheIndexEntry{
		Version:  version,
		File:     fileName,
		CachedAt: time.Now().UTC(),
	}
	l.saveCacheIndex()
}

// moduleKeyFor returns the go.mod module path that owns the given import path.
func (l *GoThirdPartyLocalLoader) moduleKeyFor(importPath string) string {
	for modPath := range l.moduleVersions {
		if importPath == modPath || strings.HasPrefix(importPath, modPath+"/") {
			return modPath
		}
	}
	return importPath
}

// findPackageSource locates the source directory for an import path.
// Checks vendor/ first, then GOMODCACHE.
func (l *GoThirdPartyLocalLoader) findPackageSource(importPath string) string {
	// 1. Check vendor/
	vendorPath := filepath.Join(l.projectRoot, "vendor", importPath)
	if hasGoFiles(vendorPath) {
		return vendorPath
	}

	// 2. Check GOMODCACHE
	modCache := os.Getenv("GOMODCACHE")
	if modCache == "" {
		gopath := os.Getenv("GOPATH")
		if gopath == "" {
			gopath = filepath.Join(os.Getenv("HOME"), "go")
		}
		modCache = filepath.Join(gopath, "pkg", "mod")
	}

	// Find the module that owns this import path
	for modPath, version := range l.moduleVersions {
		if importPath == modPath || strings.HasPrefix(importPath, modPath+"/") {
			// The subpackage path within the module
			subPkg := strings.TrimPrefix(importPath, modPath)
			modDir := filepath.Join(modCache, modPath+"@"+version)
			pkgDir := filepath.Join(modDir, subPkg)
			if hasGoFiles(pkgDir) {
				return pkgDir
			}
			// Also try without subpackage (root of the module)
			if subPkg == "" && hasGoFiles(modDir) {
				return modDir
			}
		}
	}

	return ""
}

// hasGoFiles checks if a directory exists and contains at least one non-test .go file.
func hasGoFiles(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") &&
			!strings.HasSuffix(entry.Name(), "_test.go") {
			return true
		}
	}
	return false
}

// parseGoModRequires extracts require directives from go.mod.
// Returns map of module path → version.
func parseGoModRequires(projectRoot string) map[string]string {
	goModPath := filepath.Join(projectRoot, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		return nil
	}

	requires := make(map[string]string)
	inRequireBlock := false

	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)

		if line == "require (" {
			inRequireBlock = true
			continue
		}
		if inRequireBlock && line == ")" {
			inRequireBlock = false
			continue
		}

		// Single-line require
		if strings.HasPrefix(line, "require ") && !strings.Contains(line, "(") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				requires[parts[1]] = parts[2]
			}
			continue
		}

		// Inside require block
		if inRequireBlock {
			line = strings.TrimSuffix(line, "// indirect")
			line = strings.TrimSpace(line)
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				requires[parts[0]] = parts[1]
			}
		}
	}

	return requires
}

// extractGoPackageWithTreeSitter parses .go files in a directory using tree-sitter
// and extracts exported types, methods, and functions into a GoStdlibPackage.
// After extraction, flattens embedded interface methods into parent interfaces.
func extractGoPackageWithTreeSitter(importPath, srcDir string) (*core.GoStdlibPackage, error) {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return nil, fmt.Errorf("reading directory %s: %w", srcDir, err)
	}

	pkg := core.NewGoStdlibPackage(importPath, "")

	parser := sitter.NewParser()
	parser.SetLanguage(golang.GetLanguage())
	defer parser.Close()

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") ||
			strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}

		filePath := filepath.Join(srcDir, entry.Name())
		src, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		tree, err := parser.ParseCtx(context.Background(), nil, src)
		if err != nil {
			continue
		}

		extractFromTree(tree.RootNode(), src, pkg)
		tree.Close()
	}

	// Post-processing: flatten embedded interface/struct methods into parent types.
	// e.g., if Client embeds EnqueueClient, copy EnqueueClient's methods into Client.
	flattenEmbeddedMethods(pkg)

	// Resolve cross-package embeds (e.g., io.Closer) using well-known stdlib interfaces.
	resolveWellKnownEmbeds(pkg)

	return pkg, nil
}

// resolveWellKnownEmbeds resolves cross-package embedded interfaces using a hardcoded
// table of well-known stdlib interfaces (io.Closer, io.Reader, etc.).
func resolveWellKnownEmbeds(pkg *core.GoStdlibPackage) {
	for _, typ := range pkg.Types {
		for _, embeddedName := range typ.Embeds {
			if !strings.Contains(embeddedName, ".") {
				continue // same-package — already handled
			}
			dotIdx := strings.LastIndex(embeddedName, ".")
			pkgAlias := embeddedName[:dotIdx]
			typeName := embeddedName[dotIdx+1:]

			if methods := getWellKnownInterfaceMethods(pkgAlias, typeName); methods != nil {
				for methodName, method := range methods {
					if _, exists := typ.Methods[methodName]; !exists {
						typ.Methods[methodName] = method
					}
				}
			}
		}
	}
}

// flattenEmbeddedMethods resolves embedded type references and copies their methods
// into the parent type. Handles same-package embeds (e.g., EnqueueClient) by looking
// up in pkg.Types. Cross-package embeds (e.g., io.Closer) are deferred to the loader.
func flattenEmbeddedMethods(pkg *core.GoStdlibPackage) {
	for _, typ := range pkg.Types {
		if len(typ.Embeds) == 0 {
			continue
		}

		for _, embeddedName := range typ.Embeds {
			// Same-package embed: look up directly in pkg.Types
			// e.g., "EnqueueClient" in posthog package
			bareEmbed := strings.TrimPrefix(embeddedName, "*")
			if embeddedType, ok := pkg.Types[bareEmbed]; ok {
				for methodName, method := range embeddedType.Methods {
					if _, exists := typ.Methods[methodName]; !exists {
						typ.Methods[methodName] = method
					}
				}
				// Recursively flatten (for multi-level embedding)
				if len(embeddedType.Embeds) > 0 {
					for _, deepEmbed := range embeddedType.Embeds {
						deepBare := strings.TrimPrefix(deepEmbed, "*")
						if deepType, ok2 := pkg.Types[deepBare]; ok2 {
							for methodName, method := range deepType.Methods {
								if _, exists := typ.Methods[methodName]; !exists {
									typ.Methods[methodName] = method
								}
							}
						}
					}
				}
			}
			// Cross-package embeds (e.g., "io.Closer") are resolved by resolveWellKnownEmbeds.
		}
	}
}

// getWellKnownInterfaceMethods returns methods for commonly embedded stdlib interfaces.
// This is a hardcoded fallback for when we don't have access to the StdlibLoader
// (avoiding import cycle). Covers the most security-relevant embedded interfaces.
func getWellKnownInterfaceMethods(pkg, typeName string) map[string]*core.GoStdlibFunction {
	key := pkg + "." + typeName

	wellKnown := map[string]map[string]*core.GoStdlibFunction{
		"io.Closer": {
			"Close": {Name: "Close", Returns: []*core.GoReturnValue{{Type: "error"}}, Confidence: 1.0},
		},
		"io.Reader": {
			"Read": {Name: "Read", Params: []*core.GoFunctionParam{{Name: "p", Type: "[]byte"}},
				Returns: []*core.GoReturnValue{{Name: "n", Type: "int"}, {Name: "err", Type: "error"}}, Confidence: 1.0},
		},
		"io.Writer": {
			"Write": {Name: "Write", Params: []*core.GoFunctionParam{{Name: "p", Type: "[]byte"}},
				Returns: []*core.GoReturnValue{{Name: "n", Type: "int"}, {Name: "err", Type: "error"}}, Confidence: 1.0},
		},
		"io.ReadCloser": {
			"Read":  {Name: "Read", Params: []*core.GoFunctionParam{{Name: "p", Type: "[]byte"}}, Returns: []*core.GoReturnValue{{Type: "int"}, {Type: "error"}}, Confidence: 1.0},
			"Close": {Name: "Close", Returns: []*core.GoReturnValue{{Type: "error"}}, Confidence: 1.0},
		},
		"io.WriteCloser": {
			"Write": {Name: "Write", Params: []*core.GoFunctionParam{{Name: "p", Type: "[]byte"}}, Returns: []*core.GoReturnValue{{Type: "int"}, {Type: "error"}}, Confidence: 1.0},
			"Close": {Name: "Close", Returns: []*core.GoReturnValue{{Type: "error"}}, Confidence: 1.0},
		},
		"io.ReadWriter": {
			"Read":  {Name: "Read", Params: []*core.GoFunctionParam{{Name: "p", Type: "[]byte"}}, Returns: []*core.GoReturnValue{{Type: "int"}, {Type: "error"}}, Confidence: 1.0},
			"Write": {Name: "Write", Params: []*core.GoFunctionParam{{Name: "p", Type: "[]byte"}}, Returns: []*core.GoReturnValue{{Type: "int"}, {Type: "error"}}, Confidence: 1.0},
		},
		"fmt.Stringer": {
			"String": {Name: "String", Returns: []*core.GoReturnValue{{Type: "string"}}, Confidence: 1.0},
		},
		"error": {
			"Error": {Name: "Error", Returns: []*core.GoReturnValue{{Type: "string"}}, Confidence: 1.0},
		},
	}

	return wellKnown[key]
}

// extractFromTree walks a Go AST and extracts exported declarations.
func extractFromTree(root *sitter.Node, src []byte, pkg *core.GoStdlibPackage) {
	for i := 0; i < int(root.ChildCount()); i++ {
		child := root.Child(i)
		switch child.Type() {
		case "function_declaration":
			extractFunctionDecl(child, src, pkg)
		case "method_declaration":
			extractMethodDecl(child, src, pkg)
		case "type_declaration":
			extractTypeDecl(child, src, pkg)
		}
	}
}

// extractFunctionDecl extracts an exported package-level function.
func extractFunctionDecl(node *sitter.Node, src []byte, pkg *core.GoStdlibPackage) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	name := nameNode.Content(src)
	if !isExported(name) {
		return
	}

	fn := &core.GoStdlibFunction{
		Name:       name,
		Confidence: 1.0,
	}

	// Extract parameters
	paramsNode := node.ChildByFieldName("parameters")
	if paramsNode != nil {
		fn.Params = extractParams(paramsNode, src)
		fn.Signature = fmt.Sprintf("func %s%s", name, paramsNode.Content(src))
	}

	// Extract return type
	resultNode := node.ChildByFieldName("result")
	if resultNode != nil {
		fn.Returns = extractReturns(resultNode, src)
		fn.Signature += " " + resultNode.Content(src)
	}

	pkg.Functions[name] = fn
}

// extractMethodDecl extracts an exported method on a type.
func extractMethodDecl(node *sitter.Node, src []byte, pkg *core.GoStdlibPackage) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	name := nameNode.Content(src)
	if !isExported(name) {
		return
	}

	// Extract receiver type
	receiverNode := node.ChildByFieldName("receiver")
	if receiverNode == nil {
		return
	}
	receiverType := extractReceiverTypeName(receiverNode, src)
	if receiverType == "" {
		return
	}

	fn := &core.GoStdlibFunction{
		Name:         name,
		ReceiverType: receiverType,
		Confidence:   1.0,
	}

	// Extract parameters
	paramsNode := node.ChildByFieldName("parameters")
	if paramsNode != nil {
		fn.Params = extractParams(paramsNode, src)
	}

	// Extract return type
	resultNode := node.ChildByFieldName("result")
	if resultNode != nil {
		fn.Returns = extractReturns(resultNode, src)
	}

	// Ensure type exists in package
	bareReceiver := strings.TrimPrefix(receiverType, "*")
	typ, ok := pkg.Types[bareReceiver]
	if !ok {
		typ = &core.GoStdlibType{
			Name:    bareReceiver,
			Kind:    "struct",
			Methods: make(map[string]*core.GoStdlibFunction),
		}
		pkg.Types[bareReceiver] = typ
	}
	if typ.Methods == nil {
		typ.Methods = make(map[string]*core.GoStdlibFunction)
	}
	typ.Methods[name] = fn
}

// extractTypeDecl extracts exported type declarations (struct, interface, alias).
func extractTypeDecl(node *sitter.Node, src []byte, pkg *core.GoStdlibPackage) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() != "type_spec" {
			continue
		}

		nameNode := child.ChildByFieldName("name")
		if nameNode == nil {
			continue
		}
		name := nameNode.Content(src)
		if !isExported(name) {
			continue
		}

		typeNode := child.ChildByFieldName("type")
		if typeNode == nil {
			continue
		}

		// Get or create type entry (methods may have been added already)
		typ, ok := pkg.Types[name]
		if !ok {
			typ = &core.GoStdlibType{
				Name:    name,
				Methods: make(map[string]*core.GoStdlibFunction),
			}
			pkg.Types[name] = typ
		}

		switch typeNode.Type() {
		case "struct_type":
			typ.Kind = "struct"
			typ.Fields = extractStructFields(typeNode, src)
		case "interface_type":
			typ.Kind = "interface"
			// Interface methods are part of the interface body
			extractInterfaceMethods(typeNode, src, typ)
		default:
			typ.Kind = "alias"
			typ.Underlying = typeNode.Content(src)
		}
	}
}

// extractStructFields extracts exported fields from a struct_type node.
func extractStructFields(node *sitter.Node, src []byte) []*core.GoStructField {
	var fields []*core.GoStructField

	// Find field_declaration_list
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() != "field_declaration_list" {
			continue
		}

		for j := 0; j < int(child.ChildCount()); j++ {
			field := child.Child(j)
			if field.Type() != "field_declaration" {
				continue
			}

			nameNode := field.ChildByFieldName("name")
			typeNode := field.ChildByFieldName("type")

			if typeNode == nil {
				continue
			}

			fieldName := ""
			if nameNode != nil {
				fieldName = nameNode.Content(src)
			}

			exported := fieldName == "" || isExported(fieldName) // embedded fields are exported
			if !exported {
				continue
			}

			tagNode := field.ChildByFieldName("tag")
			tag := ""
			if tagNode != nil {
				tag = tagNode.Content(src)
			}

			fields = append(fields, &core.GoStructField{
				Name:     fieldName,
				Type:     typeNode.Content(src),
				Tag:      tag,
				Exported: exported,
			})
		}
	}

	return fields
}

// extractInterfaceMethods extracts method signatures and embedded interface names
// from an interface_type node. Embedded interfaces (e.g., io.Closer, EnqueueClient)
// are recorded in typ.Embeds for post-extraction flattening.
func extractInterfaceMethods(node *sitter.Node, src []byte, typ *core.GoStdlibType) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)

		switch child.Type() {
		case "method_spec", "method_elem":
			// Direct method declaration: IsFeatureEnabled(payload string) (interface{}, error)
			extractInterfaceMethodElem(child, src, typ)

		case "type_elem":
			// Embedded type reference, wrapped in type_elem node.
			// Contains either type_identifier (same-package) or qualified_type (cross-package).
			for j := 0; j < int(child.ChildCount()); j++ {
				inner := child.Child(j)
				switch inner.Type() {
				case "type_identifier":
					typ.Embeds = append(typ.Embeds, inner.Content(src))
				case "qualified_type":
					typ.Embeds = append(typ.Embeds, inner.Content(src))
				}
			}

		case "type_identifier":
			// Embedded same-package interface (direct child, some grammars)
			typ.Embeds = append(typ.Embeds, child.Content(src))

		case "qualified_type":
			// Embedded cross-package interface (direct child, some grammars)
			typ.Embeds = append(typ.Embeds, child.Content(src))
		}
	}
}

// extractInterfaceMethodElem extracts a single method from a method_elem or method_spec node.
func extractInterfaceMethodElem(child *sitter.Node, src []byte, typ *core.GoStdlibType) {
	nameNode := child.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	name := nameNode.Content(src)
	if !isExported(name) {
		return
	}

	fn := &core.GoStdlibFunction{
		Name:       name,
		Confidence: 1.0,
	}

	paramsNode := child.ChildByFieldName("parameters")
	if paramsNode != nil {
		fn.Params = extractParams(paramsNode, src)
	}

	resultNode := child.ChildByFieldName("result")
	if resultNode != nil {
		fn.Returns = extractReturns(resultNode, src)
	}

	typ.Methods[name] = fn
}

// extractParams extracts function parameters from a parameter_list node.
func extractParams(node *sitter.Node, src []byte) []*core.GoFunctionParam {
	var params []*core.GoFunctionParam
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() != "parameter_declaration" {
			continue
		}

		typeNode := child.ChildByFieldName("type")
		if typeNode == nil {
			continue
		}
		paramType := typeNode.Content(src)

		// Check for variadic
		isVariadic := strings.HasPrefix(paramType, "...")
		if isVariadic {
			paramType = strings.TrimPrefix(paramType, "...")
		}

		// Extract parameter name(s)
		nameNode := child.ChildByFieldName("name")
		paramName := ""
		if nameNode != nil {
			paramName = nameNode.Content(src)
		}

		params = append(params, &core.GoFunctionParam{
			Name:       paramName,
			Type:       paramType,
			IsVariadic: isVariadic,
		})
	}
	return params
}

// extractReturns extracts return types from a result node.
func extractReturns(node *sitter.Node, src []byte) []*core.GoReturnValue {
	content := node.Content(src)

	// Simple return type (no parens)
	if !strings.HasPrefix(content, "(") {
		return []*core.GoReturnValue{{Type: content}}
	}

	// Multiple returns: (type1, type2, ...)
	inner := strings.TrimPrefix(content, "(")
	inner = strings.TrimSuffix(inner, ")")
	var returns []*core.GoReturnValue
	for _, part := range strings.Split(inner, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		// Handle named returns: "name type"
		fields := strings.Fields(part)
		if len(fields) == 2 {
			returns = append(returns, &core.GoReturnValue{Name: fields[0], Type: fields[1]})
		} else {
			returns = append(returns, &core.GoReturnValue{Type: part})
		}
	}
	return returns
}

// extractReceiverTypeName extracts the type name from a receiver parameter list.
// e.g., "(db *DB)" → "*DB", "(s Server)" → "Server".
func extractReceiverTypeName(node *sitter.Node, src []byte) string {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() != "parameter_declaration" {
			continue
		}
		typeNode := child.ChildByFieldName("type")
		if typeNode != nil {
			typeName := typeNode.Content(src)
			// Strip package qualifiers for receiver — we only care about the bare type name
			if strings.Contains(typeName, ".") {
				parts := strings.SplitN(typeName, ".", 2)
				typeName = parts[len(parts)-1]
			}
			return typeName
		}
	}
	return ""
}

// isExported checks if a Go identifier is exported (starts with uppercase).
func isExported(name string) bool {
	if name == "" {
		return false
	}
	return unicode.IsUpper(rune(name[0]))
}
