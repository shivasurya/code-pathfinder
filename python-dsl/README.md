# Code-Pathfinder Python DSL

Python DSL for defining security patterns in code-pathfinder.

## Installation

```bash
pip install codepathfinder
```

## Quick Start

```python
from codepathfinder import rule, calls, variable

@rule(id="code-injection", severity="critical", cwe="CWE-94")
def detect_eval():
    """Detects dangerous code execution via eval/exec"""
    return calls("eval", "exec")

@rule(id="user-input", severity="high")
def detect_user_input():
    """Detects user input variables"""
    return variable("user_input")
```

## Core Matchers

### `calls(*patterns)`

Matches function/method calls.

```python
from codepathfinder import calls

# Exact match
calls("eval")

# Multiple patterns
calls("eval", "exec", "compile")

# Wildcard patterns
calls("request.*")          # Matches request.GET, request.POST, etc.
calls("*.execute")          # Matches cursor.execute, conn.execute, etc.
```

### `variable(pattern)`

Matches variable references.

```python
from codepathfinder import variable

# Exact match
variable("user_input")

# Wildcard patterns
variable("user_*")          # Matches user_input, user_data, etc.
variable("*_id")            # Matches user_id, post_id, etc.
```

## Dataflow Analysis

### `flows(from_sources, to_sinks, sanitized_by=None, propagates_through=None, scope="global")`

Tracks tainted data flow from sources to sinks for OWASP Top 10 vulnerability detection.

```python
from codepathfinder import flows, calls, propagates

# SQL Injection
flows(
    from_sources=calls("request.GET", "request.POST"),
    to_sinks=calls("execute", "executemany"),
    sanitized_by=calls("quote_sql"),
    propagates_through=[
        propagates.assignment(),
        propagates.function_args(),
    ],
    scope="global"
)

# Command Injection
flows(
    from_sources=calls("request.POST"),
    to_sinks=calls("os.system", "subprocess.call"),
    sanitized_by=calls("shlex.quote"),
    propagates_through=[
        propagates.assignment(),
        propagates.function_args(),
        propagates.function_returns(),
    ]
)

# Path Traversal
flows(
    from_sources=calls("request.GET"),
    to_sinks=calls("open", "os.path.join"),
    sanitized_by=calls("os.path.abspath"),
    propagates_through=[propagates.assignment()],
    scope="local"
)
```

**Parameters:**
- `from_sources`: Source matcher(s) where taint originates (e.g., user input)
- `to_sinks`: Sink matcher(s) for dangerous functions
- `sanitized_by` (optional): Sanitizer matcher(s) that neutralize taint
- `propagates_through` (optional): List of propagation primitives (EXPLICIT!)
- `scope`: `"local"` (intra-procedural) or `"global"` (inter-procedural, default)

### Propagation Primitives

Propagation primitives define HOW taint flows through code:

```python
from codepathfinder import propagates

# Phase 1 (Available Now):
propagates.assignment()        # x = tainted
propagates.function_args()     # func(tainted)
propagates.function_returns()  # return tainted
```

**Important:** Propagation is EXPLICIT - you must specify which primitives to enable. No defaults are applied.

## Rule Decorator

The `@rule` decorator marks functions as security rules with metadata.

```python
from codepathfinder import rule, calls

@rule(
    id="sqli-001",
    severity="critical",
    cwe="CWE-89",
    owasp="A03:2021"
)
def detect_sql_injection():
    """Detects SQL injection vulnerabilities"""
    return calls("execute", "executemany", "raw")
```

**Parameters:**
- `id` (str): Unique rule identifier
- `severity` (str): `critical` | `high` | `medium` | `low`
- `cwe` (str, optional): CWE identifier (e.g., "CWE-89")
- `owasp` (str, optional): OWASP category (e.g., "A03:2021")

The function docstring becomes the rule description.

## JSON IR Output

Rules serialize to JSON Intermediate Representation (IR) for the Go executor:

```python
from codepathfinder import rule, calls
import json

@rule(id="test", severity="high")
def my_rule():
    return calls("eval")

# Serialize to JSON IR
ir = my_rule.execute()
print(json.dumps(ir, indent=2))
```

Output:
```json
{
  "rule": {
    "id": "test",
    "name": "my_rule",
    "severity": "high",
    "cwe": null,
    "owasp": null,
    "description": ""
  },
  "matcher": {
    "type": "call_matcher",
    "patterns": ["eval"],
    "wildcard": false,
    "match_mode": "any"
  }
}
```

## Development

```bash
# Install with dev dependencies
pip install -e ".[dev]"

# Run tests
pytest

# Format code
black codepathfinder/ tests/

# Lint
ruff check codepathfinder/ tests/

# Type check
mypy codepathfinder/
```

## Requirements

- Python 3.8+
- No external dependencies (stdlib only!)

## License

MIT
