# GAP-012: Subscript/Index Taint Propagation — Technical Report

**Branch**: `shiva/subscript-access-gpa`
**Status**: Implementation complete, ready for rule writer validation
**Date**: 2026-03-28

---

## What Changed

When Python code uses dict subscript syntax `data["key"]` instead of `data.get("key")`, the engine previously lost taint information. This fix makes three subscript patterns visible to the dataflow engine:

| Pattern | Before | After |
|---|---|---|
| `x = request.GET["key"]` | `AttributeAccess=""`, no taint source | `AttributeAccess="request.GET"` |
| `x = obj.method()["key"]` | `CallTarget='obj.method()["key"]'` (raw text) | `CallTarget="method"` (unwrapped) |
| `x = data["a"]["b"]["c"]` | Only outermost subscript inspected | Nested unwrapping to innermost value |

**No new API surface.** Rule writers use the same `attribute()` matcher. The engine now extracts the right data from subscript expressions automatically.

---

## How It Works

### The Engine Change

In `extractAssignment()` (`sast-engine/graph/callgraph/extraction/statements.go`), the RHS processing now has a dedicated `subscript` branch. When the RHS is a subscript node, the engine:

1. **Unwraps nested subscripts** — walks `value` children until it finds a non-subscript node
2. **Dispatches on the innermost value type**:
   - `attribute` → sets `AttributeAccess` (taint source matching)
   - `call` → extracts `CallTarget`, `Uses`, `CallArgs` from the inner call
   - anything else → extracts identifiers normally

```
Python code:          x = request.GET["cmd"]

Tree-sitter AST:      (assignment
                        left: (identifier) "x"
                        right: (subscript
                          value: (attribute        ← innermost value
                            object: "request"
                            attribute: "GET")
                          subscript: (string) "cmd"))

Statement output:     Def="x"
                      AttributeAccess="request.GET"  ← NEW (was empty)
                      Uses=["request"]
                      CallTarget='request.GET["cmd"]'
```

### The VDG Pipeline

The Variable Dependency Graph already matches sources against both `CallTarget` and `AttributeAccess` (since GAP-006). No VDG changes were needed:

```go
// var_dep_graph.go:73-77 — already handles both fields
if stmt.CallTarget != "" && matchesAnyPattern(stmt.CallTarget, sources) {
    node.IsTaintSrc = true
}
if stmt.AttributeAccess != "" && matchesAnyPattern(stmt.AttributeAccess, sources) {
    node.IsTaintSrc = true
}
```

---

## What Rule Writers Can Do Now

### Pattern 1: Subscript on attribute chain as taint source

Use `attribute()` to match dict subscript access on Django/Flask request objects, `os.environ`, or any attribute chain:

```python
from codepathfinder import attribute, calls, flows

@python_rule(
    id="CUSTOM-001",
    name="Django GET Parameter to Command Execution",
    severity="CRITICAL",
    cwe="CWE-78",
)
def detect_django_cmdi():
    return flows(
        from_sources=[
            attribute("request.GET"),      # matches: x = request.GET["cmd"]
            attribute("request.POST"),     # matches: x = request.POST["data"]
        ],
        to_sinks=[
            calls("subprocess.run", "subprocess.call", "subprocess.Popen"),
        ],
        scope="local",
    )
```

### Pattern 2: Flask form subscript access

```python
@python_rule(
    id="CUSTOM-002",
    name="Flask Form Subscript to Path Operation",
    severity="HIGH",
    cwe="CWE-22",
)
def detect_flask_path_traversal():
    return flows(
        from_sources=[
            attribute("flask.request.form"),   # x = flask.request.form["folder"]
            attribute("request.form"),         # x = request.form["folder"]
        ],
        to_sinks=[
            calls("os.path.join"),
        ],
        sanitized_by=[
            calls("secure_filename"),
        ],
        scope="local",
    )
```

### Pattern 3: Environment variable injection

```python
@python_rule(
    id="CUSTOM-003",
    name="Environment Variable to Command Execution",
    severity="HIGH",
    cwe="CWE-78",
)
def detect_environ_cmdi():
    return flows(
        from_sources=[
            attribute("os.environ"),    # x = os.environ["DEPLOY_CMD"]
        ],
        to_sinks=[
            calls("subprocess.run", "os.system", "os.popen"),
        ],
        sanitized_by=[
            calls("shlex.quote"),
        ],
        scope="local",
    )
```

### Pattern 4: Unmasked method call through subscript

When `obj.method()["key"]` is used, the engine now exposes the `method` as `CallTarget`. This means `calls()` matchers work through subscripts:

```python
@python_rule(
    id="CUSTOM-004",
    name="API Response Data to Command Execution",
    severity="CRITICAL",
    cwe="CWE-78",
)
def detect_api_response_cmdi():
    return flows(
        from_sources=[
            # requests.get(url).json()["data"] → CallTarget="json"
            calls("json"),
            calls("requests.get"),
        ],
        to_sinks=[
            calls("subprocess.run"),
        ],
        scope="local",
    )
```

---

## Complete Pattern Reference

### Subscript patterns that now produce `AttributeAccess`

| Python Code | `AttributeAccess` | `Uses` | Notes |
|---|---|---|---|
| `x = request.GET["key"]` | `request.GET` | `["request"]` | Django QueryDict |
| `x = request.POST["key"]` | `request.POST` | `["request"]` | Django QueryDict |
| `x = request.form["field"]` | `request.form` | `["request"]` | Flask ImmutableMultiDict |
| `x = flask.request.form["f"]` | `flask.request.form` | `["flask"]` | Fully qualified Flask |
| `x = os.environ["VAR"]` | `os.environ` | `["os"]` | Environment variables |
| `x = self.data["key"]` | `self.data` | `["data"]` | `self` filtered from Uses |
| `x = request.GET["a"]["b"]` | `request.GET` | `["request"]` | Nested subscript unwrapped |
| `x = config.db["host"]["port"]` | `config.db` | `["config"]` | Triple nesting works |

### Subscript patterns that now expose masked `CallTarget`

| Python Code | `CallTarget` | `Uses` | Notes |
|---|---|---|---|
| `x = obj.method()["key"]` | `method` | `["obj"]` | Single subscript on call |
| `x = resp.json()["data"]` | `json` | `["resp"]` | Common API pattern |
| `x = requests.get(url).json()["d"]` | `json` | `["url"]` | Chained call + subscript |
| `x = func()["key"]` | `func` | `["func"]` | Simple call + subscript |
| `x = func(a, b)["key"]` | `func` | `["a", "b"]` | Args propagated |
| `x = obj.method()["a"]["b"]` | `method` | `["obj"]` | Nested subscript on call |

### Subscript patterns that are unchanged (no `AttributeAccess`, no call unmasking)

| Python Code | `AttributeAccess` | `Uses` | Why |
|---|---|---|---|
| `x = d["key"]` | `""` | `["d"]` | Plain identifier, no attribute chain |
| `x = d[0]` | `""` | `["d"]` | Integer index |
| `x = d[idx]` | `""` | `["d", "idx"]` | Variable index in Uses |
| `x = d[1:3]` | `""` | `["d"]` | Slice notation |
| `x = "hello"[0]` | `""` | `[]` | String literal |
| `x = [1,2,3][0]` | `""` | `[]` | List literal |
| `x = {"a":1}["a"]` | `""` | `[]` | Dict literal |
| `arr[i] = value` | N/A | N/A | LHS subscript — skipped (unchanged) |

---

## Matching Behavior

The `attribute()` matcher supports exact and suffix matching:

| Pattern | Matches | Does Not Match |
|---|---|---|
| `attribute("request.GET")` | `request.GET` | `other.GET`, `GET` (no dot prefix) |
| `attribute("GET")` | `request.GET`, `django.request.GET` | `GETTING` (must be exact component) |
| `attribute("os.environ")` | `os.environ` | `os.environment` |
| `attribute("request.GET", "request.POST")` | Either one | `request.PUT` |

---

## What This Does NOT Cover

These are out of scope for GAP-012 and remain as separate gaps:

1. **Subscript in binary expressions**: `x = data["a"] + data["b"]` — the top-level RHS is a `binary_operator`, not a `subscript`. Each `data["a"]` is inside the binary op and not separately extracted. The `Uses` will include `["data"]` via recursive identifier extraction, but no `AttributeAccess` is set.

2. **Subscript on LHS**: `arr[i] = value` — deliberately skipped (doesn't define a local variable).

3. **Control flow bodies**: `if cond: x = request.GET["key"]` — statement extraction skips control flow bodies at the top level. CFG-aware analysis (Tier 1) handles this via `AnalyzeWithCFG`.

4. **Full call chain resolution**: `x = requests.get(url).json()["data"]` exposes `CallTarget="json"` but does NOT set `CallChain="requests.get.json"`. Full chain resolution requires GAP-004.

5. **Type-constrained subscript matching**: `QueryType.attr()` for subscript sources requires the type inference engine to resolve the type of the subscript's base object. This works if the type is already inferred but is not enhanced by this change.

---

## How to Test Your Rules

### Quick smoke test

```bash
# Create a test file
cat > /tmp/test_subscript.py << 'EOF'
import subprocess
import os

def cmd_view(request):
    cmd = request.GET["cmd"]
    subprocess.run(cmd, shell=True)

def env_cmd():
    val = os.environ["DEPLOY_CMD"]
    subprocess.run(val, shell=True)
EOF

# Run with your rule
./build/go/pathfinder scan --project /tmp --ruleset /path/to/your/rule.py
```

### Diagnostic check

Use the `diagnose` command to verify the engine extracts the right statement data:

```bash
./build/go/pathfinder diagnose --project /tmp
```

Look for statements where `AttributeAccess` is populated for subscript patterns.

---

## Bug Reporting

If you find a subscript pattern that should be detected but isn't, please report with:

1. **The exact Python code** (minimal reproducer)
2. **The rule you're using** (source pattern and sink pattern)
3. **Expected vs actual behavior**
4. **Output of `diagnose` command** if available

Known limitation: the engine only processes top-level statements in function bodies. Statements inside `if`/`for`/`while`/`try` blocks require CFG-aware analysis (`scope="local"` with Tier 1).

---

## Files Modified

| File | Lines Changed | Purpose |
|---|---|---|
| `sast-engine/graph/callgraph/extraction/statements.go` | +53 -14 | Core subscript extraction logic |
| `sast-engine/graph/callgraph/extraction/statements_test.go` | +282 -2 | 24 extraction tests |
| `sast-engine/dsl/attribute_matcher_test.go` | +144 | 5 DSL integration tests |
| `sast-engine/graph/callgraph/analysis/taint/var_dep_graph_test.go` | +146 | 4 VDG taint flow tests |
| `python-sdk/codepathfinder/matchers.py` | +7 -1 | Updated docstring |
| `python-sdk/tests/test_matchers.py` | +34 | 4 SDK tests |
| **Total** | **+652 -14** | |
