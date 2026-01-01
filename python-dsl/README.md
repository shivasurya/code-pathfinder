# Code-Pathfinder Python DSL

Python DSL for defining security patterns in Code Pathfinder - an open-source security suite combining structural code analysis with AI-powered vulnerability detection.

**Project Goals:**
- Real-time IDE integration bringing security insights directly into your editor
- AI-assisted analysis leveraging LLMs to understand context and identify vulnerabilities
- Unified workflow coverage from local development to CI/CD pipelines
- Flexible reporting supporting DefectDojo, GitHub Advanced Security, SARIF, and other platforms

**Documentation**: https://codepathfinder.dev/

## Installation

```bash
pip install codepathfinder
```

This installs **both** the Python DSL and the `pathfinder` CLI binary for your platform.

### Verify Installation

```bash
# Test CLI binary
pathfinder --version

# Test Python DSL
python -c "from codepathfinder import rule, calls; print('DSL OK')"
```

### Supported Platforms

- Linux (glibc): x86_64, aarch64
- macOS: arm64 (Apple Silicon), x86_64 (Intel)
- Windows: x86_64

Source distributions are available for other platforms - the binary will be downloaded automatically on first use.

## Quick Example

```python
from codepathfinder import rule, flows, calls
from codepathfinder.presets import PropagationPresets

@rule(id="sql-injection", severity="critical", cwe="CWE-89")
def detect_sql_injection():
    """Detects SQL injection vulnerabilities"""
    return flows(
        from_sources=calls("request.GET", "request.POST"),
        to_sinks=calls("execute", "executemany"),
        sanitized_by=calls("quote_sql"),
        propagates_through=PropagationPresets.standard(),
        scope="global"
    )
```

## Features

- **Matchers**: `calls()`, `variable()` for pattern matching
- **Dataflow Analysis**: `flows()` for source-to-sink taint tracking
- **Propagation**: Explicit propagation primitives (assignment, function args, returns)
- **Logic Operators**: `And()`, `Or()`, `Not()` for complex rules
- **JSON IR**: Serializes to JSON for Go executor integration

## Documentation

For detailed documentation, visit https://codepathfinder.dev/

## Requirements

- Python 3.8+
- No external dependencies (stdlib only!)

## License

AGPL-3.0 - GNU Affero General Public License v3
