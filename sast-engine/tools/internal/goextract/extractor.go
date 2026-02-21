// Package goextract provides tools for extracting Go standard library API surface
// into versioned JSON registry files for use by the Code Pathfinder SAST engine.
package goextract

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// GeneratorVersion is the version of this extraction tool.
const GeneratorVersion = "1.0.0"

// Config holds the configuration for the stdlib extractor.
type Config struct {
	// GoVersion is the Go version being extracted (e.g., "1.21", "1.26.0").
	GoVersion string
	// GOROOT is the path to the Go installation root (e.g., "/usr/local/go").
	GOROOT string
	// OutputDir is the directory where JSON registry files will be written.
	OutputDir string
}

// Extractor extracts the exported API surface of Go stdlib packages and writes
// versioned JSON registry files compatible with GoStdlibRegistry.
type Extractor struct {
	config Config
}

// NewExtractor creates a new Extractor with the given configuration.
func NewExtractor(cfg Config) *Extractor {
	return &Extractor{config: cfg}
}

// Run executes the full extraction pipeline: discover packages, extract each one,
// write per-package JSON files, and write the manifest.
func (e *Extractor) Run() error {
	if err := os.MkdirAll(e.config.OutputDir, 0o755); err != nil {
		return fmt.Errorf("creating output dir %q: %w", e.config.OutputDir, err)
	}

	importPaths, err := e.discoverPackages()
	if err != nil {
		return fmt.Errorf("discovering packages: %w", err)
	}

	sort.Strings(importPaths)

	packages := make([]*Package, 0, len(importPaths))
	filesizes := make(map[string]int64, len(importPaths))
	checksums := make(map[string]string, len(importPaths))

	for _, importPath := range importPaths {
		pkg, extractErr := e.extractPackage(importPath)
		if extractErr != nil {
			// Log and skip packages that fail to parse rather than aborting.
			fmt.Fprintf(os.Stderr, "warning: skipping %s: %v\n", importPath, extractErr)
			continue
		}
		size, checksum, writeErr := e.writePackageFile(pkg)
		if writeErr != nil {
			return fmt.Errorf("writing package file for %s: %w", importPath, writeErr)
		}
		packages = append(packages, pkg)
		filesizes[importPath] = size
		checksums[importPath] = checksum
	}

	manifest := e.buildManifest(packages, filesizes, checksums)
	if err := e.writeManifestFile(manifest); err != nil {
		return fmt.Errorf("writing manifest: %w", err)
	}

	fmt.Printf("Extracted %d packages → %s\n", len(packages), e.config.OutputDir)
	return nil
}

// discoverPackages walks $GOROOT/src and returns all public package import paths.
func (e *Extractor) discoverPackages() ([]string, error) {
	srcDir := filepath.Join(e.config.GOROOT, "src")
	var packages []string

	err := filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !d.IsDir() {
			return nil
		}
		// Skip the root itself.
		if path == srcDir {
			return nil
		}
		// Skip directories whose name should be excluded.
		if shouldSkipComponent(d.Name()) {
			return filepath.SkipDir
		}
		hasGo, err := dirHasGoFiles(path)
		if err != nil {
			return err
		}
		if hasGo {
			// filepath.Rel cannot fail here: path is always a descendant of srcDir.
			relPath, _ := filepath.Rel(srcDir, path)
			packages = append(packages, filepath.ToSlash(relPath))
		}
		return nil
	})

	return packages, err
}

// shouldSkipComponent returns true for directory names that represent non-public
// or toolchain-internal parts of $GOROOT/src.
func shouldSkipComponent(name string) bool {
	switch name {
	case "internal", "cmd", "testdata", "vendor", "builtin":
		return true
	}
	// Skip hidden and underscore-prefixed directories.
	if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
		return true
	}
	return false
}

// dirHasGoFiles reports whether a directory contains at least one non-test .go file.
func dirHasGoFiles(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go") {
			return true, nil
		}
	}
	return false, nil
}

// extractPackage parses all non-test .go files in the given import path and
// returns the extracted Package metadata.
func (e *Extractor) extractPackage(importPath string) (*Package, error) {
	dir := filepath.Join(e.config.GOROOT, "src", filepath.FromSlash(importPath))

	fset := token.NewFileSet()
	pkgMap, err := parser.ParseDir(fset, dir, func(fi fs.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", importPath, err)
	}

	pkg := &Package{
		ImportPath:  importPath,
		GoVersion:   e.config.GoVersion,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Functions:   make(map[string]*Function),
		Types:       make(map[string]*Type),
		Constants:   make(map[string]*Constant),
		Variables:   make(map[string]*Variable),
	}

	for pkgName, astPkg := range pkgMap {
		// Skip test-only packages.
		if strings.HasSuffix(pkgName, "_test") {
			continue
		}
		for _, file := range astPkg.Files {
			e.extractFromFile(pkg, file, fset)
		}
	}

	return pkg, nil
}

// extractFromFile extracts exported declarations from a single parsed file.
func (e *Extractor) extractFromFile(pkg *Package, file *ast.File, fset *token.FileSet) {
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if !d.Name.IsExported() {
				continue
			}
			fn := extractFuncDecl(d, fset)
			if d.Recv != nil && len(d.Recv.List) > 0 {
				// Method: attach to receiver type (Methods is always non-nil from extractTypeSpec).
				recvType := receiverTypeName(d.Recv.List[0].Type)
				if t, ok := pkg.Types[recvType]; ok {
					t.Methods[fn.Name] = fn
				}
				// Also add as a top-level function with receiver prefix.
				pkg.Functions[recvType+"."+fn.Name] = fn
			} else {
				pkg.Functions[fn.Name] = fn
			}
		case *ast.GenDecl:
			switch d.Tok {
			case token.TYPE:
				for _, spec := range d.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok || !ts.Name.IsExported() {
						continue
					}
					t := extractTypeSpec(ts, d.Doc, fset)
					pkg.Types[t.Name] = t
				}
			case token.CONST:
				consts := extractConsts(d, fset)
				for _, c := range consts {
					pkg.Constants[c.Name] = c
				}
			case token.VAR:
				vars := extractVars(d, fset)
				for _, v := range vars {
					pkg.Variables[v.Name] = v
				}
			}
		}
	}
}

// extractFuncDecl converts an *ast.FuncDecl to a Function.
func extractFuncDecl(decl *ast.FuncDecl, fset *token.FileSet) *Function {
	params, isVariadic := extractParams(decl.Type.Params, fset)
	returns := extractReturns(decl.Type.Results, fset)
	typeParams := extractTypeParams(decl.Type.TypeParams, fset)

	recvType := ""
	if decl.Recv != nil && len(decl.Recv.List) > 0 {
		recvType = exprToString(decl.Recv.List[0].Type, fset)
	}

	sig := funcSignatureStr(decl, fset)
	deprecated, deprecatedMsg := isDeprecated(decl.Doc)

	return &Function{
		Name:          decl.Name.Name,
		Signature:     sig,
		Params:        params,
		Returns:       returns,
		IsVariadic:    isVariadic,
		IsGeneric:     len(typeParams) > 0,
		TypeParams:    typeParams,
		ReceiverType:  recvType,
		Confidence:    1.0,
		Docstring:     extractDocstring(decl.Doc),
		Deprecated:    deprecated,
		DeprecatedMsg: deprecatedMsg,
	}
}

// extractParams extracts function parameters and reports whether the last is variadic.
func extractParams(fields *ast.FieldList, fset *token.FileSet) ([]*FunctionParam, bool) {
	if fields == nil || len(fields.List) == 0 {
		return nil, false
	}

	var params []*FunctionParam
	isVariadic := false

	for i, field := range fields.List {
		typStr := exprToString(field.Type, fset)
		variadic := false

		if _, ok := field.Type.(*ast.Ellipsis); ok {
			variadic = true
			// Unwrap "...T" → "T" in the type string.
			if ellipsis, ok2 := field.Type.(*ast.Ellipsis); ok2 {
				typStr = exprToString(ellipsis.Elt, fset)
			}
			// The last parameter is variadic.
			if i == len(fields.List)-1 {
				isVariadic = true
			}
		}

		if len(field.Names) == 0 {
			// Unnamed parameter.
			params = append(params, &FunctionParam{
				Name:       "",
				Type:       typStr,
				IsVariadic: variadic,
			})
		} else {
			for _, name := range field.Names {
				params = append(params, &FunctionParam{
					Name:       name.Name,
					Type:       typStr,
					IsVariadic: variadic,
				})
			}
		}
	}

	return params, isVariadic
}

// extractReturns extracts function return values from a field list.
func extractReturns(fields *ast.FieldList, fset *token.FileSet) []*ReturnValue {
	if fields == nil || len(fields.List) == 0 {
		return nil
	}

	var returns []*ReturnValue
	for _, field := range fields.List {
		typStr := exprToString(field.Type, fset)
		if len(field.Names) == 0 {
			returns = append(returns, &ReturnValue{Type: typStr})
		} else {
			for _, name := range field.Names {
				returns = append(returns, &ReturnValue{Type: typStr, Name: name.Name})
			}
		}
	}

	return returns
}

// extractTypeParams extracts generic type parameters from a field list.
func extractTypeParams(fields *ast.FieldList, fset *token.FileSet) []*TypeParam {
	if fields == nil || len(fields.List) == 0 {
		return nil
	}

	var params []*TypeParam
	for _, field := range fields.List {
		constraint := exprToString(field.Type, fset)
		for _, name := range field.Names {
			params = append(params, &TypeParam{
				Name:       name.Name,
				Constraint: constraint,
			})
		}
	}

	return params
}

// extractTypeSpec converts an *ast.TypeSpec to a Type.
func extractTypeSpec(spec *ast.TypeSpec, groupDoc *ast.CommentGroup, fset *token.FileSet) *Type {
	doc := spec.Doc
	if doc == nil {
		doc = groupDoc
	}

	typeParams := extractTypeParams(spec.TypeParams, fset)

	t := &Type{
		Name:       spec.Name.Name,
		Methods:    make(map[string]*Function),
		IsGeneric:  len(typeParams) > 0,
		TypeParams: typeParams,
		Docstring:  extractDocstring(doc),
	}

	switch st := spec.Type.(type) {
	case *ast.StructType:
		t.Kind = "struct"
		t.Fields = extractStructFields(st, fset)
	case *ast.InterfaceType:
		t.Kind = "interface"
		t.Methods = extractInterfaceMethods(st, fset)
	default:
		t.Kind = "alias"
		t.Underlying = exprToString(spec.Type, fset)
	}

	return t
}

// extractStructFields extracts exported fields from a struct type.
func extractStructFields(st *ast.StructType, fset *token.FileSet) []*StructField {
	if st == nil || st.Fields == nil {
		return nil
	}

	var fields []*StructField
	for _, field := range st.Fields.List {
		typStr := exprToString(field.Type, fset)
		tagStr := ""
		if field.Tag != nil {
			tagStr = strings.Trim(field.Tag.Value, "`")
		}

		if len(field.Names) == 0 {
			// Embedded (anonymous) field.
			name := embeddedFieldName(field.Type)
			if name == "" || !ast.IsExported(name) {
				continue
			}
			fields = append(fields, &StructField{
				Name:     name,
				Type:     typStr,
				Tag:      tagStr,
				Exported: true,
			})
		} else {
			for _, name := range field.Names {
				if !name.IsExported() {
					continue
				}
				fields = append(fields, &StructField{
					Name:     name.Name,
					Type:     typStr,
					Tag:      tagStr,
					Exported: true,
				})
			}
		}
	}

	return fields
}

// embeddedFieldName returns the type name used as an embedded field name.
func embeddedFieldName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		return embeddedFieldName(e.X)
	case *ast.SelectorExpr:
		return e.Sel.Name
	}
	return ""
}

// extractInterfaceMethods extracts exported method signatures from an interface type.
func extractInterfaceMethods(iface *ast.InterfaceType, fset *token.FileSet) map[string]*Function {
	methods := make(map[string]*Function)
	if iface == nil || iface.Methods == nil {
		return methods
	}

	for _, method := range iface.Methods.List {
		if len(method.Names) == 0 {
			// Embedded interface — skip to avoid recursive resolution.
			continue
		}
		for _, name := range method.Names {
			if !name.IsExported() {
				continue
			}
			// In valid Go, named interface members always have *ast.FuncType.
			funcType := method.Type.(*ast.FuncType) //nolint:forcetypeassert
			params, isVariadic := extractParams(funcType.Params, fset)
			returns := extractReturns(funcType.Results, fset)
			typeParams := extractTypeParams(funcType.TypeParams, fset)

			var sigBuf bytes.Buffer
			_ = printer.Fprint(&sigBuf, fset, funcType)

			fn := &Function{
				Name:       name.Name,
				Signature:  name.Name + " " + sigBuf.String(),
				Params:     params,
				Returns:    returns,
				IsVariadic: isVariadic,
				IsGeneric:  len(typeParams) > 0,
				TypeParams: typeParams,
				Confidence: 1.0,
				Docstring:  extractDocstring(method.Comment),
			}
			methods[name.Name] = fn
		}
	}

	return methods
}

// extractConsts extracts exported constant declarations from a GenDecl.
func extractConsts(decl *ast.GenDecl, fset *token.FileSet) []*Constant {
	var consts []*Constant
	for _, spec := range decl.Specs {
		// Per Go spec, CONST GenDecl specs are always *ast.ValueSpec.
		vs := spec.(*ast.ValueSpec) //nolint:forcetypeassert
		typStr := ""
		if vs.Type != nil {
			typStr = exprToString(vs.Type, fset)
		}
		doc := vs.Doc
		if doc == nil {
			doc = decl.Doc
		}
		for i, name := range vs.Names {
			if !name.IsExported() {
				continue
			}
			value := ""
			isIota := false
			if i < len(vs.Values) {
				value = exprToString(vs.Values[i], fset)
				isIota = containsIota(vs.Values[i])
			} else if len(vs.Values) == 0 && decl.Tok == token.CONST {
				// Continuation of iota sequence when value is omitted.
				isIota = true
			}
			consts = append(consts, &Constant{
				Name:       name.Name,
				Type:       typStr,
				Value:      value,
				IsIota:     isIota,
				Confidence: 1.0,
				Docstring:  extractDocstring(doc),
			})
		}
	}
	return consts
}

// containsIota reports whether an expression references the iota identifier.
func containsIota(expr ast.Expr) bool {
	if expr == nil {
		return false
	}
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name == "iota"
	case *ast.BinaryExpr:
		return containsIota(e.X) || containsIota(e.Y)
	case *ast.UnaryExpr:
		return containsIota(e.X)
	case *ast.CallExpr:
		for _, arg := range e.Args {
			if containsIota(arg) {
				return true
			}
		}
	case *ast.ParenExpr:
		return containsIota(e.X)
	}
	return false
}

// extractVars extracts exported variable declarations from a GenDecl.
func extractVars(decl *ast.GenDecl, fset *token.FileSet) []*Variable {
	var vars []*Variable
	for _, spec := range decl.Specs {
		// Per Go spec, VAR GenDecl specs are always *ast.ValueSpec.
		vs := spec.(*ast.ValueSpec) //nolint:forcetypeassert
		typStr := ""
		if vs.Type != nil {
			typStr = exprToString(vs.Type, fset)
		}
		doc := vs.Doc
		if doc == nil {
			doc = decl.Doc
		}
		for _, name := range vs.Names {
			if !name.IsExported() {
				continue
			}
			vars = append(vars, &Variable{
				Name:       name.Name,
				Type:       typStr,
				Confidence: 1.0,
				Docstring:  extractDocstring(doc),
			})
		}
	}
	return vars
}

// buildManifest constructs a Manifest from the list of extracted packages.
func (e *Extractor) buildManifest(packages []*Package, filesizes map[string]int64, checksums map[string]string) *Manifest {
	stats := &RegistryStats{}
	entries := make([]*PackageEntry, 0, len(packages))

	for _, pkg := range packages {
		hasGenerics := false
		for _, fn := range pkg.Functions {
			if fn.IsGeneric {
				hasGenerics = true
				break
			}
		}
		if !hasGenerics {
			for _, t := range pkg.Types {
				if t.IsGeneric {
					hasGenerics = true
					break
				}
			}
		}

		entry := &PackageEntry{
			ImportPath:    pkg.ImportPath,
			Checksum:      checksums[pkg.ImportPath],
			FileSize:      filesizes[pkg.ImportPath],
			FunctionCount: len(pkg.Functions),
			TypeCount:     len(pkg.Types),
			ConstantCount: len(pkg.Constants),
		}
		entries = append(entries, entry)

		stats.TotalPackages++
		stats.TotalFunctions += len(pkg.Functions)
		stats.TotalTypes += len(pkg.Types)
		stats.TotalConstants += len(pkg.Constants)
		if hasGenerics {
			stats.PackagesWithGenerics++
		}
	}

	// Sort entries alphabetically for deterministic manifest output.
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].ImportPath < entries[j].ImportPath
	})

	return &Manifest{
		SchemaVersion:    "1.0.0",
		RegistryVersion:  "v1",
		GoVersion:        parseGoVersion(e.config.GoVersion),
		GeneratedAt:      time.Now().UTC().Format(time.RFC3339),
		GeneratorVersion: GeneratorVersion,
		BaseURL: fmt.Sprintf(
			"https://assets.codepathfinder.dev/registries/go%s/stdlib/v1",
			e.config.GoVersion,
		),
		Packages:   entries,
		Statistics: stats,
	}
}

// writePackageFile marshals pkg to JSON, writes it to OutputDir, and returns
// (fileSize, sha256Checksum, error).
// Note: json.MarshalIndent cannot fail for the Package type (no channels, funcs, or cycles).
func (e *Extractor) writePackageFile(pkg *Package) (int64, string, error) {
	// Ignore marshal error: Package contains only JSON-serializable types.
	data, _ := json.MarshalIndent(pkg, "", "  ") //nolint:errchkjson

	filename := packageFileName(pkg.ImportPath)
	path := filepath.Join(e.config.OutputDir, filename)

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return 0, "", fmt.Errorf("writing %s: %w", path, err)
	}

	return int64(len(data)), checksumBytes(data), nil
}

// writeManifestFile marshals manifest to JSON and writes it to OutputDir/manifest.json.
// Note: json.MarshalIndent cannot fail for the Manifest type (no channels, funcs, or cycles).
func (e *Extractor) writeManifestFile(manifest *Manifest) error {
	// Ignore marshal error: Manifest contains only JSON-serializable types.
	data, _ := json.MarshalIndent(manifest, "", "  ") //nolint:errchkjson

	path := filepath.Join(e.config.OutputDir, "manifest.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing manifest: %w", err)
	}
	return nil
}

// --- Helper functions ---

// exprToString converts an AST expression to its source representation.
// printer.Fprint never returns an error when writing to a bytes.Buffer.
func exprToString(expr ast.Expr, fset *token.FileSet) string {
	if expr == nil {
		return ""
	}
	var buf bytes.Buffer
	_ = printer.Fprint(&buf, fset, expr)
	return buf.String()
}

// funcSignatureStr returns the text of a function declaration without its body.
func funcSignatureStr(decl *ast.FuncDecl, fset *token.FileSet) string {
	// Temporarily nil the body to get just the signature.
	body := decl.Body
	decl.Body = nil
	var buf bytes.Buffer
	_ = printer.Fprint(&buf, fset, decl)
	decl.Body = body
	return strings.TrimSpace(buf.String())
}

// receiverTypeName returns the base type name of a method receiver expression.
func receiverTypeName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.StarExpr:
		return receiverTypeName(e.X)
	case *ast.Ident:
		return e.Name
	case *ast.IndexExpr:
		// Generic receiver: T[A]
		return receiverTypeName(e.X)
	case *ast.IndexListExpr:
		// Generic receiver: T[A, B]
		return receiverTypeName(e.X)
	}
	return ""
}

// extractDocstring returns the cleaned text of a comment group, truncated at 500 chars.
func extractDocstring(doc *ast.CommentGroup) string {
	if doc == nil {
		return ""
	}
	text := strings.TrimSpace(doc.Text())
	if len(text) > 500 {
		text = text[:500] + "..."
	}
	return text
}

// isDeprecated checks for a "Deprecated:" line in a doc comment.
func isDeprecated(doc *ast.CommentGroup) (bool, string) {
	if doc == nil {
		return false, ""
	}
	for _, line := range strings.Split(doc.Text(), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "Deprecated:") {
			msg := strings.TrimSpace(strings.TrimPrefix(trimmed, "Deprecated:"))
			return true, msg
		}
	}
	return false, ""
}

// packageFileName converts an import path to a safe JSON filename.
// e.g., "net/http" → "net_http_stdlib.json".
func packageFileName(importPath string) string {
	safe := strings.ReplaceAll(importPath, "/", "_")
	return safe + "_stdlib.json"
}

// checksumBytes returns a "sha256:<hex>" checksum of the given data.
func checksumBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// parseGoVersion parses a version string like "1.21" or "1.21.0" into a VersionInfo.
func parseGoVersion(version string) VersionInfo {
	parts := strings.Split(version, ".")
	major, minor, patch := 1, 0, 0

	if len(parts) >= 1 {
		major, _ = strconv.Atoi(parts[0])
	}
	if len(parts) >= 2 {
		minor, _ = strconv.Atoi(parts[1])
	}
	if len(parts) >= 3 {
		patch, _ = strconv.Atoi(parts[2])
	}

	full := version
	if len(parts) == 2 {
		full = version + ".0"
	}

	return VersionInfo{
		Major: major,
		Minor: minor,
		Patch: patch,
		Full:  full,
	}
}
