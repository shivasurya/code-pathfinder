# Pathfinder Python DSL

Python DSL for defining security patterns in code-pathfinder.

## Installation

```bash
pip install codepathfinder
```

## Quick Start

```python
from pathfinder import rule, calls, variable

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
from pathfinder import calls

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
from pathfinder import variable

# Exact match
variable("user_input")

# Wildcard patterns
variable("user_*")          # Matches user_input, user_data, etc.
variable("*_id")            # Matches user_id, post_id, etc.
```

## Rule Decorator

The `@rule` decorator marks functions as security rules with metadata.

```python
from pathfinder import rule, calls

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
from pathfinder import rule, calls
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
black pathfinder/ tests/

# Lint
ruff check pathfinder/ tests/

# Type check
mypy pathfinder/
```

## Requirements

- Python 3.8+
- No external dependencies (stdlib only!)

## License

MIT
