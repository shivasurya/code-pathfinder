package registry

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode"

	sitter "github.com/smacker/go-tree-sitter"
	golang "github.com/smacker/go-tree-sitter/golang"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/output"
)

// GoThirdPartyLocalLoader extracts type metadata from third-party Go packages
// found in vendor/ or GOMODCACHE. Uses tree-sitter for lightweight parsing.
// Implements core.GoThirdPartyLoader.
type GoThirdPartyLocalLoader struct {
	projectRoot    string
	moduleVersions map[string]string                 // import path → version (from go.mod require)
	packageCache   map[string]*core.GoStdlibPackage  // import path → extracted package
	cacheMutex     sync.RWMutex
	logger         *output.Logger
}

// NewGoThirdPartyLocalLoader creates a loader that finds and parses third-party
// Go packages from vendor/ or GOMODCACHE.
func NewGoThirdPartyLocalLoader(projectRoot string, logger *output.Logger) *GoThirdPartyLocalLoader {
	loader := &GoThirdPartyLocalLoader{
		projectRoot:    projectRoot,
		moduleVersions: make(map[string]string),
		packageCache:   make(map[string]*core.GoStdlibPackage),
		logger:         logger,
	}
	loader.moduleVersions = parseGoModRequires(projectRoot)
	if logger != nil {
		logger.Debug("Go third-party local loader: found %d dependencies in go.mod", len(loader.moduleVersions))
	}
	return loader
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

// getOrLoadPackage retrieves a package from cache or parses it from source.
func (l *GoThirdPartyLocalLoader) getOrLoadPackage(importPath string) (*core.GoStdlibPackage, error) {
	// Fast path: check cache
	l.cacheMutex.RLock()
	if pkg, ok := l.packageCache[importPath]; ok {
		l.cacheMutex.RUnlock()
		return pkg, nil
	}
	l.cacheMutex.RUnlock()

	// Slow path: find and parse
	l.cacheMutex.Lock()
	defer l.cacheMutex.Unlock()

	// Double-check
	if pkg, ok := l.packageCache[importPath]; ok {
		return pkg, nil
	}

	srcDir := l.findPackageSource(importPath)
	if srcDir == "" {
		l.packageCache[importPath] = nil // cache negative result
		return nil, nil
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
	return pkg, nil
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

	return pkg, nil
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

// extractInterfaceMethods extracts method signatures from an interface_type node.
func extractInterfaceMethods(node *sitter.Node, src []byte, typ *core.GoStdlibType) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		// Look for method_spec nodes inside the interface body
		if child.Type() == "method_spec" || child.Type() == "method_elem" {
			nameNode := child.ChildByFieldName("name")
			if nameNode == nil {
				continue
			}
			name := nameNode.Content(src)
			if !isExported(name) {
				continue
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
	}
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
// e.g., "(db *DB)" → "*DB", "(s Server)" → "Server"
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
