<div align="center">
  <img src="./assets/banner.png" alt="Code Pathfinder - Open-source SAST with cross-file dataflow analysis" width="100%">
</div>

<div align="center">

<h3>Open-source SAST engine that traces vulnerabilities across files and functions</h3>

[Website](https://codepathfinder.dev/) · [Docs](https://codepathfinder.dev/docs/quickstart) · [Rule Registry](https://codepathfinder.dev/registry) · [MCP Server](https://codepathfinder.dev/mcp) · [Blog](https://codepathfinder.dev/blog)

[![Build](https://github.com/shivasurya/code-pathfinder/actions/workflows/build.yml/badge.svg)](https://github.com/shivasurya/code-pathfinder/actions/workflows/build.yml)
[![GitHub Release](https://img.shields.io/github/v/release/shivasurya/code-pathfinder?label=release)](https://github.com/shivasurya/code-pathfinder/releases)
[![Apache-2.0 License](https://img.shields.io/badge/license-Apache--2.0-blue)](https://github.com/shivasurya/code-pathfinder/blob/main/LICENSE)
[![GitHub Stars](https://img.shields.io/github/stars/shivasurya/code-pathfinder?style=flat)](https://github.com/shivasurya/code-pathfinder/stargazers)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/shivasurya/code-pathfinder)

</div>

---

## Quick Start

**Install:**

```bash
brew install shivasurya/tap/pathfinder
```

**Scan a Python project** (rules download automatically):

```bash
pathfinder scan --ruleset python/all --project .
```

**Scan Dockerfiles:**

```bash
pathfinder scan --ruleset docker/all --project .
```

No config files, no API keys, no cloud accounts. Results in your terminal in seconds.

---

<!-- TODO: Add demo video/GIF here -->

## What is Code Pathfinder?

Code Pathfinder is an open-source static analysis engine that builds a graph of your codebase and traces how data flows through it. It parses source code into Abstract Syntax Trees, constructs call graphs across files, and runs taint analysis to find source-to-sink vulnerabilities that span multiple files and function boundaries.

**v2.0** introduces **cross-file dataflow analysis**: trace user input from an HTTP handler in one file through helper functions and into a SQL query in another file. This is the kind of analysis that pattern-matching tools miss entirely.

### Cross-File Taint Analysis

Most open-source SAST tools operate on single files. Code Pathfinder v2.0 tracks tainted data across file boundaries:

```
app.py:5    user_input = request.get("query")     ← Source: user-controlled input
  ↓ calls
db.py:12    cursor.execute(query)                  ← Sink: SQL execution
```

The engine builds a Variable Dependency Graph (VDG) per function, then connects them through inter-procedural taint transfer summaries. When `user_input` flows into a function parameter in another file, the taint propagates through the call graph to the sink.

### How It Works

```
Source Code → Tree-sitter AST → Call Graph → Variable Dependency Graph → Taint Analysis → Findings
                                     ↓
                              Inter-procedural
                              Taint Summaries
                              (cross-file flows)
```

1. **Parse**: Tree-sitter builds ASTs for Python, Dockerfiles, and Docker Compose files
2. **Index**: Extract functions, call sites, parameters, and assignments into a queryable call graph
3. **Analyze**: Build VDGs per function, resolve inter-procedural flows, run taint analysis
4. **Detect**: Python-based security rules query the graph to find source-to-sink paths
5. **Report**: Output findings as text, JSON, SARIF (GitHub Code Scanning), or CSV

## 190 Security Rules, Ready to Use

Rules download from CDN automatically. No need to clone the repo or manage rule files.

| Language | Bundles | Rules | Coverage |
|----------|---------|-------|----------|
| **[Python](https://codepathfinder.dev/registry/python)** | django, flask, aws_lambda, cryptography, jwt, lang, deserialization, pyramid | 158 | SQL injection, RCE, SSRF, path traversal, XSS, deserialization, crypto misuse, JWT vulnerabilities |
| **[Docker](https://codepathfinder.dev/registry/docker)** | security, best-practice, performance | 37 | Root user, exposed secrets, image pinning, multi-stage builds, layer optimization |
| **[Docker Compose](https://codepathfinder.dev/registry/docker-compose)** | security, networking | 10 | Privileged mode, socket exposure, capability escalation, network isolation |

```bash
# Scan with a specific bundle
pathfinder scan --ruleset python/django --project .

# Scan with multiple bundles
pathfinder scan --ruleset python/flask --ruleset python/jwt --project .

# Scan a single rule
pathfinder scan --ruleset python/PYTHON-DJANGO-SEC-001 --project .

# Scan all rules for a language
pathfinder scan --ruleset python/all --project .
```

Browse all rules with examples and test cases at the [Rule Registry](https://codepathfinder.dev/registry).

## MCP Server for AI Coding Assistants

Code Pathfinder runs as an [MCP server](https://codepathfinder.dev/mcp), giving Claude Code, Cursor, Cline, and other AI assistants access to call graphs, data flows, and security analysis. More context than LSP, focused on security and code structure.

```bash
pathfinder serve --project .
```

The MCP server exposes tools for querying the code graph: find callers/callees, trace data flows, search for patterns, and run security rules — all available to the AI assistant during code review or development.

## Write Custom Rules

Security rules are Python scripts using the [PathFinder SDK](https://codepathfinder.dev/docs/rules). Define sources, sinks, and sanitizers — the dataflow engine handles the analysis.

Here's a real rule from the repo ([`PYTHON-DJANGO-SEC-001`](./rules/python/django/PYTHON-DJANGO-SEC-001/rule.py)) that detects SQL injection in Django:

```python
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class DBCursor(QueryType):
    fqns = ["sqlite3.Cursor", "psycopg2.extensions.cursor"]
    match_subclasses = True

@python_rule(
    id="PYTHON-DJANGO-SEC-001",
    name="Django SQL Injection via cursor.execute()",
    severity="CRITICAL",
    cwe="CWE-89",
)
def detect_django_cursor_sqli():
    return flows(
        from_sources=[
            calls("request.GET.get"),
            calls("request.POST.get"),
        ],
        to_sinks=[
            DBCursor.method("execute").tracks(0),
            calls("cursor.execute"),
        ],
        sanitized_by=[calls("escape"), calls("escape_string")],
        propagates_through=PropagationPresets.standard(),
        scope="global",  # cross-file taint analysis
    )
```

```bash
# Run your custom rules
pathfinder scan --rules ./my_rules/ --project .
```

Explore all 190 rules in the [`rules/`](./rules) directory or browse the [Rule Registry](https://codepathfinder.dev/registry). See the [rule writing guide](https://codepathfinder.dev/docs/rules) and [dataflow documentation](https://codepathfinder.dev/docs/dataflow) to write your own.

See the [rule writing guide](https://codepathfinder.dev/docs/rules) and [dataflow documentation](https://codepathfinder.dev/docs/dataflow) for more.

## Installation

### Homebrew (Recommended)

```bash
brew install shivasurya/tap/pathfinder
```

### pip

Installs the CLI binary and Python SDK for writing rules.

```bash
pip install codepathfinder
```

### Docker

```bash
docker pull shivasurya/code-pathfinder:stable-latest

docker run --rm -v "$(pwd):/src" \
  shivasurya/code-pathfinder:stable-latest \
  scan --ruleset python/all --project /src
```

### Pre-Built Binaries

Download from [GitHub Releases](https://github.com/shivasurya/code-pathfinder/releases) for Linux (amd64, arm64), macOS (Intel, Apple Silicon), and Windows (x64).

### From Source

```bash
git clone https://github.com/shivasurya/code-pathfinder
cd code-pathfinder/sast-engine
gradle buildGo
./build/go/pathfinder --help
```

## Usage

```bash
# Scan with text output (default)
pathfinder scan --ruleset python/all --project .

# JSON output
pathfinder scan --ruleset python/all --project . --output json --output-file results.json

# SARIF output (GitHub Code Scanning)
pathfinder scan --ruleset python/all --project . --output sarif --output-file results.sarif

# CSV output
pathfinder scan --ruleset python/all --project . --output csv --output-file results.csv

# Fail CI on critical/high findings
pathfinder scan --ruleset python/all --project . --fail-on=critical,high

# MCP server mode
pathfinder serve --project .

# Verbose output with statistics
pathfinder scan --ruleset python/all --project . --verbose
```

## GitHub Action

```yaml
name: Code Pathfinder Security SAST Scan

on:
  pull_request:

permissions:
  security-events: write
  contents: read
  pull-requests: write

jobs:
  security-scan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
        with:
          fetch-depth: 0

      - name: Run Security Scan
        uses: shivasurya/code-pathfinder@v2.0.0
        with:
          ruleset: python/all, docker/all, docker-compose/all
          verbose: true
          pr-comment: ${{ github.event_name == 'pull_request' }}
          pr-inline: ${{ github.event_name == 'pull_request' }}
          github-token: ${{ secrets.GITHUB_TOKEN }}

      - name: Upload SARIF
        uses: github/codeql-action/upload-sarif@v4
        if: always()
        with:
          sarif_file: pathfinder-results.sarif
```

See the full example: [`.github/workflows/code-pathfinder-scan.yml`](.github/workflows/code-pathfinder-scan.yml)

<details>
<summary><strong>Action Inputs</strong></summary>

| Input | Description | Default |
|-------|-------------|---------|
| `rules` | Path to local Python rule files or directory | - |
| `ruleset` | Remote ruleset(s), comma-separated (e.g., `python/all`, `docker/security`) | - |
| `project` | Path to source code | `.` |
| `output` | Output format: `sarif`, `json`, or `csv` | `sarif` |
| `output-file` | Output file path | `pathfinder-results.sarif` |
| `fail-on` | Fail on severities (e.g., `critical,high`) | - |
| `verbose` | Enable verbose output | `false` |
| `debug` | Enable debug diagnostics with timestamps | `false` |
| `skip-tests` | Skip test files | `true` |
| `refresh-rules` | Force refresh cached rulesets | `false` |
| `disable-metrics` | Disable anonymous usage metrics | `false` |
| `python-version` | Python version to use | `3.12` |
| `pr-comment` | Post summary comment on pull request | `false` |
| `pr-inline` | Post inline review comments for critical/high findings | `false` |
| `github-token` | GitHub token (required when `pr-comment` or `pr-inline` is enabled) | - |
| `no-diff` | Disable diff-aware scanning (scan all files) | `false` |

Either `rules` or `ruleset` is required.

</details>

## Supported Languages

| Language | Analysis | Status |
|----------|----------|--------|
| **Python** | Cross-file dataflow, taint analysis, call graphs | Stable |
| **Dockerfile** | Instruction analysis, security patterns | Stable |
| **Docker Compose** | Configuration analysis, security patterns | Stable |
| **Go** | AST analysis, call graphs | Coming soon |

## Contributing

Contributions are welcome. Read the [Contributing Guide](./CONTRIBUTING.md) for setup instructions, how to run tests locally, and the PR process.

All contributors must sign the [Contributor License Agreement (CLA)](./CLA.md) before any pull request can be merged.

- [Report bugs or request features](https://github.com/shivasurya/code-pathfinder/issues)
- [Ask questions or start a discussion](https://github.com/shivasurya/code-pathfinder/discussions)
- [Write security rules](https://codepathfinder.dev/docs/rules)

## License

[Apache-2.0](https://github.com/shivasurya/code-pathfinder/blob/main/LICENSE)
