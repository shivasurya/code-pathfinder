package builder

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTypeInference_StdlibChaining proves stdlib return-type chaining
// resolves the full chain: sqlite3.connect() → Connection → .cursor() → Cursor → .execute().
// Requires CDN data regenerated with typeshed overlay (PR-00) for C builtin return types.
func TestTypeInference_StdlibChaining(t *testing.T) {
	projectPath, err := filepath.Abs("../../../test-fixtures/python/stdlib_chaining")
	require.NoError(t, err)

	codeGraph := graph.Initialize(projectPath, nil)
	require.NotNil(t, codeGraph)

	logger := output.NewLogger(output.VerbosityDefault)
	callGraph, _, err := BuildCallGraphFromPath(codeGraph, projectPath, logger)
	require.NoError(t, err)

	// Evidence: collect FQNs for all call sites in the search function.
	type evidence struct {
		target    string
		fqn       string
		resolved  bool
		typeInfer bool
	}
	var searchEvidence []evidence

	for _, sites := range callGraph.CallSites {
		for _, cs := range sites {
			if cs.Location.File != "" && strings.Contains(cs.Location.File, "app.py") {
				searchEvidence = append(searchEvidence, evidence{
					target:    cs.Target,
					fqn:       cs.TargetFQN,
					resolved:  cs.Resolved,
					typeInfer: cs.ResolvedViaTypeInference,
				})
			}
		}
	}

	t.Log("=== STDLIB TYPE INFERENCE EVIDENCE ===")
	for _, e := range searchEvidence {
		t.Logf("  target=%-30s fqn=%-40s resolved=%v typeInfer=%v", e.target, e.fqn, e.resolved, e.typeInfer)
	}

	// Assertion 1: cursor.execute → sqlite3.Cursor.execute (type-inferred FQN).
	foundCursorExecute := false
	for _, e := range searchEvidence {
		if e.target == "cursor.execute" && e.fqn == "sqlite3.Cursor.execute" {
			foundCursorExecute = true
		}
	}
	assert.True(t, foundCursorExecute,
		"cursor.execute must resolve to sqlite3.Cursor.execute via type inference chain")

	// Assertion 2: conn.cursor → sqlite3.Connection.cursor (type-inferred FQN).
	foundConnCursor := false
	for _, e := range searchEvidence {
		if e.target == "conn.cursor" && e.fqn == "sqlite3.Connection.cursor" {
			foundConnCursor = true
		}
	}
	assert.True(t, foundConnCursor,
		"conn.cursor must resolve to sqlite3.Connection.cursor via type inference chain")

	// Assertion 3: sqlite3.connect → sqlite3.connect (direct import FQN).
	foundConnect := false
	for _, e := range searchEvidence {
		if e.target == "sqlite3.connect" && e.fqn == "sqlite3.connect" {
			foundConnect = true
		}
	}
	assert.True(t, foundConnect,
		"sqlite3.connect must resolve to sqlite3.connect via import resolution")

	// Assertion 4: hashlib.md5 → hashlib.md5 (direct stdlib FQN).
	foundMd5 := false
	for _, e := range searchEvidence {
		if e.target == "hashlib.md5" && e.fqn == "hashlib.md5" {
			foundMd5 = true
		}
	}
	assert.True(t, foundMd5,
		"hashlib.md5 must resolve to hashlib.md5")

	// Assertion 5: os.chmod → os.chmod (direct stdlib FQN).
	foundChmod := false
	for _, e := range searchEvidence {
		if e.target == "os.chmod" && e.fqn == "os.chmod" {
			foundChmod = true
		}
	}
	assert.True(t, foundChmod,
		"os.chmod must resolve to os.chmod")

	// Assertion 6: cursor.fetchall → sqlite3.Cursor.fetchall (chained type inference).
	foundFetchall := false
	for _, e := range searchEvidence {
		if e.target == "cursor.fetchall" && e.fqn == "sqlite3.Cursor.fetchall" {
			foundFetchall = true
		}
	}
	assert.True(t, foundFetchall,
		"cursor.fetchall must resolve to sqlite3.Cursor.fetchall via type inference chain")
}

// TestTypeInference_ThirdPartyChaining proves third-party return-type chaining
// resolves: requests.get() → Response → .json() → dict.
func TestTypeInference_ThirdPartyChaining(t *testing.T) {
	t.Skip("Requires third-party CDN data (requests, flask) — skip until CDN is available in CI")
	projectPath, err := filepath.Abs("../../../test-fixtures/python/stdlib_chaining")
	require.NoError(t, err)

	codeGraph := graph.Initialize(projectPath, nil)
	require.NotNil(t, codeGraph)

	logger := output.NewLogger(output.VerbosityDefault)
	callGraph, _, err := BuildCallGraphFromPath(codeGraph, projectPath, logger)
	require.NoError(t, err)

	type evidence struct {
		target string
		fqn    string
	}
	var thirdPartyEvidence []evidence

	for _, sites := range callGraph.CallSites {
		for _, cs := range sites {
			if cs.Location.File != "" && strings.Contains(cs.Location.File, "app_thirdparty.py") {
				thirdPartyEvidence = append(thirdPartyEvidence, evidence{
					target: cs.Target,
					fqn:    cs.TargetFQN,
				})
			}
		}
	}

	t.Log("=== THIRD-PARTY TYPE INFERENCE EVIDENCE ===")
	for _, e := range thirdPartyEvidence {
		t.Logf("  target=%-30s fqn=%-40s", e.target, e.fqn)
	}

	// Assertion: requests.get → requests.get or api.get (import resolved).
	foundRequestsGet := false
	for _, e := range thirdPartyEvidence {
		if e.target == "requests.get" && strings.Contains(e.fqn, "requests") {
			foundRequestsGet = true
		}
	}
	assert.True(t, foundRequestsGet,
		"requests.get must resolve to an FQN containing 'requests'")

	// Assertion: requests.post → similar resolution.
	foundRequestsPost := false
	for _, e := range thirdPartyEvidence {
		if e.target == "requests.post" && strings.Contains(e.fqn, "requests") {
			foundRequestsPost = true
		}
	}
	assert.True(t, foundRequestsPost,
		"requests.post must resolve to an FQN containing 'requests'")

	// Assertion: resp.json (on requests.Response) resolves with type info.
	foundRespJson := false
	for _, e := range thirdPartyEvidence {
		if e.target == "resp.json" && e.fqn != "" {
			foundRespJson = true
			t.Logf("  resp.json resolved to: %s", e.fqn)
		}
	}
	assert.True(t, foundRespJson,
		"resp.json() must resolve (proving third-party return-type chaining)")

	// Assertion: Flask request.args.get resolved.
	foundFlaskRequest := false
	for _, e := range thirdPartyEvidence {
		if e.target == "request.args.get" && strings.Contains(e.fqn, "flask") {
			foundFlaskRequest = true
		}
	}
	assert.True(t, foundFlaskRequest,
		"request.args.get must resolve to an FQN containing 'flask'")
}
