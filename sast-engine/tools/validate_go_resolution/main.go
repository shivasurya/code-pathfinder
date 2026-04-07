// validate_go_resolution extracts ground-truth Go call sites from a project using
// the Go type checker (go/packages) and optionally compares them against pathfinder's
// resolution output to compute precision/recall statistics.
//
// Usage:
//
//	# Extract ground truth only
//	go run ./tools/validate_go_resolution/ --project /path/to/project --pkg ./some/pkg/... --out ground_truth.jsonl
//
//	# Compare pathfinder output against ground truth
//	go run ./tools/validate_go_resolution/ --project /path/to/project --pkg ./server/... \
//	  --pathfinder callsites.jsonl --out ground_truth.jsonl
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

// groundTruthRecord is one call site record from the type checker.
type groundTruthRecord struct {
	File      string `json:"file"`
	Line      int    `json:"line"`
	Col       int    `json:"col"`
	CalleeFQN string `json:"callee_fqn"` // e.g., "net/http.Client.Do"
	Kind      string `json:"kind"`       // "method" or "func"
}

// pathfinderRecord mirrors the callSiteRecord from resolution_report.go.
type pathfinderRecord struct {
	File       string `json:"file"`
	Line       int    `json:"line"`
	Col        int    `json:"col"`
	CallerFQN  string `json:"caller_fqn"`
	Target     string `json:"target"`
	OurFQN     string `json:"our_fqn"`
	Resolved   bool   `json:"resolved"`
	TypeSource string `json:"type_source,omitempty"`
	IsStdlib   bool   `json:"is_stdlib,omitempty"`
}

func main() {
	projectDir := flag.String("project", "", "Go project root directory (required)")
	pkgPattern := flag.String("pkg", "./...", "Package pattern to analyze (e.g., ./server/...)")
	outFile := flag.String("out", "ground_truth.jsonl", "Output file for ground truth records")
	pfFile := flag.String("pathfinder", "", "Pathfinder callsites JSONL to compare against")
	flag.Parse()

	if *projectDir == "" {
		log.Fatal("--project is required")
	}

	absProject, err := filepath.Abs(*projectDir)
	if err != nil {
		log.Fatalf("resolving project dir: %v", err)
	}

	fmt.Printf("Loading packages from %s (pattern: %s)...\n", absProject, *pkgPattern)

	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedSyntax |
			packages.NeedTypes |
			packages.NeedTypesInfo |
			packages.NeedImports,
		Dir: absProject,
	}

	pkgs, err := packages.Load(cfg, *pkgPattern)
	if err != nil {
		log.Fatalf("loading packages: %v", err)
	}

	var loadErrors []string
	for _, pkg := range pkgs {
		for _, e := range pkg.Errors {
			loadErrors = append(loadErrors, fmt.Sprintf("  %s: %s", pkg.PkgPath, e))
		}
	}
	if len(loadErrors) > 0 {
		fmt.Fprintf(os.Stderr, "Package load warnings (%d):\n", len(loadErrors))
		for _, e := range loadErrors[:min(10, len(loadErrors))] {
			fmt.Fprintln(os.Stderr, e)
		}
	}
	fmt.Printf("Loaded %d packages\n", len(pkgs))

	// Extract ground truth call sites
	records := extractCallSites(pkgs)
	fmt.Printf("Extracted %d method/function call sites\n", len(records))

	// Write ground truth output
	if err := writeJSONL(records, *outFile); err != nil {
		log.Fatalf("writing output: %v", err)
	}
	fmt.Printf("Ground truth written to %s\n", *outFile)

	// If pathfinder output provided, compare
	if *pfFile != "" {
		pfRecords, err := readPathfinderRecords(*pfFile)
		if err != nil {
			log.Fatalf("reading pathfinder records: %v", err)
		}
		compare(records, pfRecords, absProject)
	}
}

// extractCallSites walks all loaded packages and extracts method/function call sites
// with their ground-truth callee FQNs from the type checker.
func extractCallSites(pkgs []*packages.Package) []groundTruthRecord {
	var records []groundTruthRecord
	seen := make(map[string]bool) // deduplicate by file:line:col

	for _, pkg := range pkgs {
		if pkg.TypesInfo == nil {
			continue
		}
		fset := pkg.Fset

		for _, file := range pkg.Syntax {
			ast.Inspect(file, func(n ast.Node) bool {
				callExpr, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}

				pos := fset.Position(callExpr.Pos())
				key := fmt.Sprintf("%s:%d:%d", pos.Filename, pos.Line, pos.Column)
				if seen[key] {
					return true
				}

				// Method call: obj.Method(...)
				if sel, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
					if selection, ok := pkg.TypesInfo.Selections[sel]; ok {
						rec := buildMethodRecord(selection, pos, fset)
						if rec != nil {
							seen[key] = true
							records = append(records, *rec)
							return true
						}
					}
				}

				// Package-level function call: pkg.Func(...) or Func(...)
				var ident *ast.Ident
				switch fun := callExpr.Fun.(type) {
				case *ast.Ident:
					ident = fun
				case *ast.SelectorExpr:
					ident = fun.Sel
				}
				if ident != nil {
					if obj, ok := pkg.TypesInfo.Uses[ident]; ok {
						if fn, ok := obj.(*types.Func); ok && fn.Pkg() != nil {
							calleeFQN := fn.Pkg().Path() + "." + fn.Name()
							seen[key] = true
							records = append(records, groundTruthRecord{
								File:      pos.Filename,
								Line:      pos.Line,
								Col:       pos.Column,
								CalleeFQN: calleeFQN,
								Kind:      "func",
							})
						}
					}
				}

				return true
			})
		}
	}

	return records
}

// buildMethodRecord constructs a groundTruthRecord for a method call selection.
func buildMethodRecord(sel *types.Selection, pos token.Position, _ *token.FileSet) *groundTruthRecord {
	obj := sel.Obj()
	if obj == nil || obj.Pkg() == nil {
		return nil
	}

	fn, ok := obj.(*types.Func)
	if !ok {
		return nil
	}

	// Extract the receiver (concrete) type name
	recv := sel.Recv()
	typeName := extractTypeName(recv)

	pkgPath := fn.Pkg().Path()
	var calleeFQN string
	if typeName != "" {
		calleeFQN = pkgPath + "." + typeName + "." + fn.Name()
	} else {
		calleeFQN = pkgPath + "." + fn.Name()
	}

	return &groundTruthRecord{
		File:      pos.Filename,
		Line:      pos.Line,
		Col:       pos.Column,
		CalleeFQN: calleeFQN,
		Kind:      "method",
	}
}

// extractTypeName gets the base type name from a types.Type (stripping pointer qualifiers).
func extractTypeName(t types.Type) string {
	switch tt := t.(type) {
	case *types.Pointer:
		return extractTypeName(tt.Elem())
	case *types.Named:
		return tt.Obj().Name()
	case *types.Interface:
		if tt.NumMethods() == 0 {
			return "" // empty interface — skip
		}
		// For named interfaces, we'd need the outer Named wrapper; return empty for anonymous
		return ""
	default:
		return ""
	}
}

// compare performs a precision/recall analysis between ground truth and pathfinder output.
func compare(gtRecords []groundTruthRecord, pfRecords []pathfinderRecord, projectRoot string) {
	// Index ground truth by file:line
	gtByLine := make(map[string][]groundTruthRecord)
	for _, r := range gtRecords {
		key := fmt.Sprintf("%s:%d", r.File, r.Line)
		gtByLine[key] = append(gtByLine[key], r)
	}

	// Index pathfinder resolved records by file:line
	pfByLine := make(map[string][]pathfinderRecord)
	for _, r := range pfRecords {
		if !r.Resolved {
			continue
		}
		key := fmt.Sprintf("%s:%d", r.File, r.Line)
		pfByLine[key] = append(pfByLine[key], r)
	}

	var (
		totalPFResolved    int
		matched            int   // pf_fqn matches gt_fqn
		mismatched         int   // pf says resolved, gt disagrees on target
		noGroundTruth      int   // pf resolved but go/packages has no record at that line
		mismatchExamples   []mismatchRecord
	)

	for key, pfList := range pfByLine {
		for _, pf := range pfList {
			totalPFResolved++
			gtList := gtByLine[key]
			if len(gtList) == 0 {
				noGroundTruth++
				continue
			}

			// Try to find a matching GT record (normalize FQNs for comparison)
			ourNorm := normalizeFQN(pf.OurFQN)
			found := false
			for _, gt := range gtList {
				gtNorm := normalizeFQN(gt.CalleeFQN)
				if ourNorm == gtNorm {
					found = true
					break
				}
			}

			if found {
				matched++
			} else {
				mismatched++
				if len(mismatchExamples) < 50 {
					mismatchExamples = append(mismatchExamples, mismatchRecord{
						File:      relativePath(pf.File, projectRoot),
						Line:      pf.Line,
						Target:    pf.Target,
						OurFQN:    pf.OurFQN,
						TrueFQNs:  collectFQNs(gtList),
						Source:    pf.TypeSource,
					})
				}
			}
		}
	}

	// Count ground truth calls that pathfinder missed (false negatives / unresolved)
	pfResolvedKeys := make(map[string]bool)
	for key := range pfByLine {
		pfResolvedKeys[key] = true
	}
	missedByPF := 0
	for key := range gtByLine {
		if !pfResolvedKeys[key] {
			missedByPF++
		}
	}

	comparable := totalPFResolved - noGroundTruth
	precision := 0.0
	if comparable > 0 {
		precision = float64(matched) / float64(comparable) * 100.0
	}

	fmt.Println("\n=== Validation Results ===")
	fmt.Printf("Pathfinder resolved calls:     %d\n", totalPFResolved)
	fmt.Printf("  With ground truth at line:   %d\n", comparable)
	fmt.Printf("  No ground truth at line:     %d (package-level calls, non-method, etc.)\n", noGroundTruth)
	fmt.Printf("\nOf the comparable %d calls:\n", comparable)
	fmt.Printf("  Correct (matched GT):        %d\n", matched)
	fmt.Printf("  Wrong target (mismatch):     %d\n", mismatched)
	fmt.Printf("\nPrecision:                     %.1f%%\n", precision)
	fmt.Printf("\nGround truth calls pathfinder missed: %d\n", missedByPF)

	if len(mismatchExamples) > 0 {
		fmt.Printf("\n=== Top Mismatches (up to 50) ===\n")
		// Sort by source to group patterns
		sort.Slice(mismatchExamples, func(i, j int) bool {
			return mismatchExamples[i].Source < mismatchExamples[j].Source
		})

		// Group by source
		sourceGroups := make(map[string]int)
		for _, m := range mismatchExamples {
			sourceGroups[m.Source]++
		}
		fmt.Println("Mismatch by type_source:")
		for src, count := range sourceGroups {
			if src == "" {
				src = "(traditional/import)"
			}
			fmt.Printf("  %-35s %d\n", src, count)
		}

		fmt.Printf("\nSample mismatches:\n")
		shown := 0
		for _, m := range mismatchExamples {
			if shown >= 20 {
				break
			}
			src := m.Source
			if src == "" {
				src = "traditional"
			}
			fmt.Printf("  %s:%d  target=%q  [%s]\n", m.File, m.Line, m.Target, src)
			fmt.Printf("    ours: %s\n", m.OurFQN)
			fmt.Printf("    gt:   %s\n", strings.Join(m.TrueFQNs, " | "))
			shown++
		}
	}
}

type mismatchRecord struct {
	File     string
	Line     int
	Target   string
	OurFQN   string
	TrueFQNs []string
	Source   string
}

func collectFQNs(records []groundTruthRecord) []string {
	out := make([]string, len(records))
	for i, r := range records {
		out[i] = r.CalleeFQN
	}
	return out
}

// normalizeFQN strips pointer markers and normalizes the FQN for comparison.
// pathfinder:    "net/http.Request.FormValue"
// go/packages:   "net/http.Request.FormValue"  (our format already matches)
func normalizeFQN(fqn string) string {
	// Strip leading "*"
	fqn = strings.TrimPrefix(fqn, "*")
	// Both systems should now use "pkgPath.TypeName.MethodName"
	return fqn
}

func relativePath(abs, root string) string {
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return abs
	}
	return rel
}

func writeJSONL(records []groundTruthRecord, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, r := range records {
		if err := enc.Encode(r); err != nil {
			return err
		}
	}
	return nil
}

func readPathfinderRecords(path string) ([]pathfinderRecord, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var records []pathfinderRecord
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var rec pathfinderRecord
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			return nil, fmt.Errorf("parsing line: %w", err)
		}
		records = append(records, rec)
	}
	return records, scanner.Err()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
