# Bug: Relative Import FQN Mismatch in Deep Attribute Chain Method Lookup

**Component**: `sast-engine/graph/callgraph/resolution/attribute.go` + `extraction/attributes.go`
**Function**: `resolveMethodOnType()` (line 170) — fails due to FQN constructed from attribute registry not matching callgraph key
**Introduced**: Commit `e601234` (deep chain resolution) — chain walk works, final step fails on relative imports
**Severity**: HIGH — blocks L1 type-inferred source matching for projects using relative imports
**Reproduces on**: pyload (confirmed), any project using `from .submodule import Class` pattern
**Does NOT reproduce on**: flat multi-file projects, projects using absolute imports

---

## Summary

After commit `e601234`, deep attribute chains like `self.pyload.config.get()` correctly resolve the intermediate types through the chain walk (Step 2). The chain `self → pyload → config` resolves to type `pyload.config.parser.ConfigParser`. However, the final method lookup (Step 3) fails because `resolveMethodOnType` constructs `pyload.config.parser.ConfigParser.get` and checks `callGraph.Functions`, which stores the key as `pyload.core.config.parser.ConfigParser.get`.

The root cause is a **relative import FQN resolution mismatch**:
- **Attribute registry** stores type as `pyload.config.parser.ConfigParser` (resolved from relative import `from .config.parser import ConfigParser` — the `.` is resolved relative to the importing module's parent, dropping the `core` segment)
- **Callgraph** stores function as `pyload.core.config.parser.ConfigParser.get` (derived from the actual file path `pyload/core/config/parser.py`)

This does NOT reproduce on flat projects or absolute imports because both systems agree on the FQN. It only triggers when the class is imported via Python relative imports (`from .x.y import Z`).

---

## Reproduction

**NOTE**: This bug does NOT reproduce on flat multi-file projects with absolute imports.
It ONLY reproduces when relative imports are used (`from .submodule import Class`).

### Scenario A: Flat project — WORKS (no bug)

```python
# /tmp/test-flat/config.py
class Config:
    def __init__(self):
        self.data = {}
    def get(self, section, key):
        return self.data.get(section, {}).get(key, "")

# /tmp/test-flat/core.py
from config import Config
class Core:
    def __init__(self):
        self.config = Config()

# /tmp/test-flat/manager.py
import subprocess
from core import Core
class Manager:
    def __init__(self):
        self.core = Core()
    def run_command(self):
        cmd = self.core.config.get("commands", "startup")
        subprocess.run(cmd)
```

```bash
pathfinder scan -r RULE -p /tmp/test-flat/ --skip-tests=false
# Result: DETECTED — self.core.config.get() resolves to config.Config.get ✅
# Both attribute registry and callgraph use "config.Config.get"
```

### Scenario B: Package with relative imports — FAILS (the bug)

```python
# /tmp/test-pkg/myapp/__init__.py
(empty)

# /tmp/test-pkg/myapp/config/parser.py
class ConfigParser:
    def __init__(self, userdir):
        self.config = {}
    def get(self, section, option):
        return self.config[section][option]["value"]

# /tmp/test-pkg/myapp/core/__init__.py
from ..config.parser import ConfigParser    # ← RELATIVE IMPORT
class Core:
    def __init__(self, userdir):
        self.config = ConfigParser(userdir)

# /tmp/test-pkg/myapp/managers/thread_manager.py
import subprocess
class ThreadManager:
    def __init__(self, core):
        self.core = core
    def reconnect(self):
        script = self.core.config.get("reconnect", "script")
        subprocess.run(script)
```

```bash
pathfinder scan -r RULE -p /tmp/test-pkg/ --skip-tests=false --debug
# Result: 0 findings ❌
# Debug shows:
#   method get not found on type config.parser.ConfigParser
#
# Attribute registry FQN: config.parser.ConfigParser  (from relative import resolution)
# Callgraph function FQN: myapp.config.parser.ConfigParser.get  (from file path)
# MISMATCH: "config.parser.ConfigParser.get" != "myapp.config.parser.ConfigParser.get"
```

### Scenario C: Real-world CVE (pyload) — CONFIRMED BUG

```bash
git clone --depth 1 https://github.com/pyload/pyload.git /tmp/pyload
pathfinder scan -r RULE -p /tmp/pyload/src/ --skip-tests=false --debug
```

The relevant pyload code:

```python
# pyload/core/__init__.py:117
from .config.parser import ConfigParser    # ← RELATIVE IMPORT (dot prefix)

# pyload/core/__init__.py:124
self.config = ConfigParser(self.userdir)

# pyload/core/managers/thread_manager.py:176
reconnect_script = self.pyload.config.get("reconnect", "script")
# ↑ 3-level chain: self → pyload → config → get()

# pyload/core/managers/thread_manager.py:199
subprocess.run(reconnect_script)  # ← CVE GHSA-r7mc-x6x7-cqxx
```

Debug output confirms:
```
Deep chains (3+ levels):   0 (0.0%)     ← chains ARE resolving (was 756 before fix)

Custom class samples:
  - method get not found on type pyload.config.parser.ConfigParser
  - method set not found on type pyload.config.parser.ConfigParser
  - method save not found on type pyload.config.parser.ConfigParser
```

**The mismatch**:
- Attribute registry resolves `self.config = ConfigParser(...)` where `ConfigParser` was imported via `from .config.parser import ConfigParser`
- The `.config.parser` relative import resolves to `pyload.config.parser` (parent of `core/__init__.py` is `pyload/`, then append `config.parser`)
- But the actual file is at `pyload/core/config/parser.py`, so the callgraph stores it as `pyload.core.config.parser.ConfigParser.get`
- **Missing `core` segment**: `pyload.config.parser` vs `pyload.core.config.parser`

---

## Root Cause Analysis

### The failing code path

**File**: `sast-engine/graph/callgraph/resolution/attribute.go`, lines 195-209

```go
func resolveMethodOnType(
    typeFQN string,     // "pyload.config.parser.ConfigParser"
    methodName string,  // "get"
    ...
) (string, bool, *core.TypeInfo) {
    // Lines 177-193: Builtin check — skipped (not builtins.*)

    // Line 196: Construct method FQN
    methodFQN := typeFQN + "." + methodName
    // → "pyload.config.parser.ConfigParser.get"

    // Lines 198-209: Check callgraph
    if callGraph != nil {
        if node := callGraph.Functions[methodFQN]; node != nil {
            // → node is NIL — method not found
            return methodFQN, true, &core.TypeInfo{...}
        }
    }

    // Line 213: Falls through to failure
    attributeFailureStats.CustomClassUnsupported++
    // → "method get not found on type pyload.config.parser.ConfigParser"
    return "", false, nil
}
```

### The FQN mismatch (CONFIRMED)

Verified on pyload's codebase:

| System | FQN for ConfigParser | Source |
|--------|---------------------|--------|
| **Attribute registry** | `pyload.config.parser.ConfigParser` | Resolved from relative import `from .config.parser import ConfigParser` in `pyload/core/__init__.py` |
| **Callgraph** | `pyload.core.config.parser.ConfigParser` | Derived from file path `pyload/core/config/parser.py` |

The relative import `from .config.parser import ConfigParser`:
- Is in file `pyload/core/__init__.py`
- The `.` resolves relative to the **parent package** of `core/`, which is `pyload/`
- So `.config.parser` → `pyload.config.parser` (going up from `core/` to `pyload/`, then down to `config/parser`)
- But the actual file is `pyload/core/config/parser.py` → FQN should be `pyload.core.config.parser`
- **The `core` segment is lost during relative import resolution**

This is a bug in how the attribute extractor resolves relative import paths to FQNs. It resolves `.config.parser` as `{parent_package}.config.parser` but the parent package is computed incorrectly — it goes one level too high.

### How to diagnose

Add temporary debug logging to `resolveMethodOnType` to print:
1. The constructed `methodFQN`
2. All keys in `callGraph.Functions` that contain the class name

```go
// Temporary debug — add before line 198
if strings.Contains(typeFQN, "Config") {
    log.Printf("[DEBUG] Looking for method %q on type %q", methodName, typeFQN)
    log.Printf("[DEBUG] Constructed FQN: %q", methodFQN)
    for key := range callGraph.Functions {
        if strings.Contains(key, "ConfigParser") && strings.Contains(key, "get") {
            log.Printf("[DEBUG] Callgraph has: %q", key)
        }
    }
}
```

This will immediately reveal the FQN format stored in the callgraph vs what's constructed.

---

## Likely Fixes

### Fix A (RECOMMENDED): Fix relative import FQN resolution in attribute extractor

The root cause is in how `resolveClassNameForChain()` or the attribute extraction resolves `from .config.parser import ConfigParser`. The `.` prefix should resolve relative to the **current package** (`pyload.core`), producing `pyload.core.config.parser.ConfigParser`. Instead, it resolves relative to the **parent package** (`pyload`), producing `pyload.config.parser.ConfigParser`.

The fix should align relative import resolution with Python's actual semantics:
- `from .x import Y` in `pyload/core/__init__.py` → `pyload.core.x.Y` (same package)
- `from ..x import Y` in `pyload/core/__init__.py` → `pyload.x.Y` (parent package)

**Where to look**:
- `extraction/attributes.go` — where `class:ConfigParser` placeholder is created
- `resolution/attribute.go:resolveClassNameForChain()` — where placeholder is resolved to FQN
- The ImportMap construction — how `from .config.parser import ConfigParser` is recorded

**Pros**: Correct fix at the source, all downstream consumers get correct FQNs
**Cons**: Need to understand how ImportMap handles relative imports

### Fix B: Suffix-based fallback in resolveMethodOnType

If the exact `callGraph.Functions[methodFQN]` lookup fails, try suffix matching:

```go
// After exact lookup fails (line 209), try suffix matching
if callGraph != nil {
    // Extract "ConfigParser.get" from "pyload.config.parser.ConfigParser.get"
    suffix := extractClassAndMethod(typeFQN, methodName)
    for fqn, node := range callGraph.Functions {
        if strings.HasSuffix(fqn, "."+suffix) &&
           (node.Type == "method" || node.Type == "function_definition") {
            return fqn, true, &core.TypeInfo{
                TypeFQN:    typeFQN,
                Confidence: float32(attrConfidence * 0.85), // slight confidence penalty
                Source:     "self_attribute_custom_class_fuzzy",
            }
        }
    }
}
```

**Pros**: Quick fix, handles any FQN mismatch without fixing root cause
**Cons**: O(n) scan of callgraph on every miss, could match wrong class if names collide (e.g., `app.Config.get` vs `lib.Config.get`). Better as a temporary workaround.

### Fix C: Build a class→method index keyed by short class name

During callgraph construction, build a reverse index:
```go
type ClassMethodIndex map[string]map[string]string
// shortClassName → methodName → full function FQN
// "ConfigParser" → "get" → "pyload.core.config.parser.ConfigParser.get"

// During callgraph build:
for fqn, node := range callGraph.Functions {
    if node.Type == "method" {
        className := extractShortClassName(fqn) // "ConfigParser"
        methodName := extractMethodName(fqn)     // "get"
        index[className][methodName] = fqn
    }
}
```

Then `resolveMethodOnType` extracts the short class name from the type FQN and uses this index.

**Pros**: O(1) lookup, tolerates any FQN format difference
**Cons**: Ambiguous if two classes share the same short name (need disambiguation by module proximity)

---

## Verification

### Test 1: Simple project (above)
```bash
pathfinder scan -r /tmp/test-rule -p /tmp/test-project/ --skip-tests=false
# Expected after fix: 1 finding at manager.py:12
```

### Test 2: Pyload CVE (GHSA-r7mc-x6x7-cqxx)
```bash
# L1 rule with ConfigParserType.method("get") as source
pathfinder scan -r /tmp/test-rule-pyload -p /tmp/pyload/src/ --skip-tests=false
# Expected after fix: 1 finding at thread_manager.py:199
# Currently: 0 findings (method lookup fails)
```

### Test 3: Existing tests must pass
```bash
cd sast-engine && go test ./...
# All existing tests must still pass
```

### Test 4: Verify no regressions on builtin types
```bash
# self.data.get() where data is builtins.dict should still resolve
# self.session.execute() where session is sqlite3.Connection should still resolve
```

---

## Impact

This is the **last blocker for L1 type-inferred source matching** on project-internal classes. The chain walk (Step 2) is fixed. The method lookup (Step 3) is the remaining issue.

| What | Status |
|------|--------|
| Deep chain walk (self.a.b.c) | Fixed in `e601234` |
| Intermediate type resolution | Working (resolves to correct class FQN) |
| Method lookup on builtins | Working (dict.get, list.append etc.) |
| Method lookup on stdlib | Working (via CDN registry) |
| **Method lookup on project classes** | **BROKEN** (FQN mismatch) |

Once fixed, the following CVE rules would achieve **L1** (both source and sink type-inferred):

| Rule | Source | Sink | Current | After Fix |
|------|--------|------|---------|-----------|
| SEC-111 | `ConfigParser.method("get")` | `subprocess.method("run")` | L2 | **L1** |
| SEC-138 | Graph query sources | `Neo4jDriver.method("run")` | L4 | **L2+** |
| Any rule with project-internal class sources | — | — | Blocked | **Unblocked** |

---

## Files to Inspect

| File | What to check |
|------|--------------|
| `resolution/attribute.go:196-209` | `resolveMethodOnType` — the failing lookup |
| `extraction/attributes.go` | How `ClassFQN` is set in the attribute registry |
| `builder/builder.go` | How function FQNs are set in `callGraph.Functions` |
| `core/callgraph.go` | `Functions` map key format |

The fix is likely small — it's a string format mismatch between two subsystems that were built independently and now need to agree on FQN conventions.
