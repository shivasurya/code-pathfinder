<div align="center">
  <img src="./assets/banner.png" alt="Code Pathfinder - AI-Native static code analysis security scanner" width="100%">
</div>

<div align="center">

[Website](https://codepathfinder.dev/) • [Installation](https://codepathfinder.dev/docs/quickstart) • [Rule Registry](https://codepathfinder.dev/registry) • [How to write rule?](https://codepathfinder.dev/docs/rules) • [VS Code](https://marketplace.visualstudio.com/items?itemName=codepathfinder.secureflow) • [Open VSX](https://open-vsx.org/extension/codepathfinder/secureflow)

[![Build](https://github.com/shivasurya/code-pathfinder/actions/workflows/build.yml/badge.svg)](https://github.com/shivasurya/code-pathfinder/actions/workflows/build.yml)
[![VS Code Marketplace](https://img.shields.io/visual-studio-marketplace/v/codepathfinder.secureflow?label=VS%20Code&logo=visualstudiocode)](https://marketplace.visualstudio.com/items?itemName=codepathfinder.secureflow)
[![Open VSX](https://img.shields.io/open-vsx/v/codepathfinder/secureflow?label=Open%20VSX&logo=vscodium)](https://open-vsx.org/extension/codepathfinder/secureflow)
[![AGPL-3.0 License](https://img.shields.io/github/license/shivasurya/code-pathfinder)](https://github.com/shivasurya/code-pathfinder/blob/main/LICENSE)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/shivasurya/code-pathfinder)

</div>

# [Code Pathfinder](https://codepathfinder.dev)

AI-Native static code analysis for modern security teams.

Code Pathfinder is an open-source security scanner that builds a queryable graph of your codebase. It parses code into Abstract Syntax Trees (AST), builds Control Flow Graphs (CFG) to track execution paths, and constructs Data Flow Graphs (DFG) to trace how data moves through your application. Instead of regex pattern matching per language, it indexes the entire codebase as structured data and lets you write queries that trace data flows across Python, [Dockerfiles](https://codepathfinder.dev/registry), and [docker-compose](https://codepathfinder.dev/blog/announcing-docker-compose-security-rules) files in a single rule.

**Use it for:**
- **CVE detection and vulnerability research**: Understand how dependencies are used, what privileges they run with, and what attack surface they expose
- **[MCP server](https://codepathfinder.dev/mcp) for AI coding assistants**: Provides code intelligence to Claude, GPT, and other AI assistants - more context than LSP, focused on security and data flow
- **In-editor security checks**: Catch vulnerable patterns as you write code in VS Code
- **CI/CD pipelines**: Automated security scanning with SARIF output for GitHub Advanced Security, DefectDojo integration
- **Custom security rules**: Write project-specific rules in Python to detect patterns that matter to your team

## What it does

- **Structural analysis**: Builds call graphs, dataflow graphs, and taint tracking to [find exploit paths](https://codepathfinder.dev/blog/static-analysis-isnt-enough-understanding-library-interactions-for-effective-data-flow-tracking) through your code, not just pattern matches.
- **AI-powered triage**: [SecureFlow](https://codepathfinder.dev/secureflow-ai) runs LLMs (Claude, GPT, Gemini, Grok, Ollama, etc.) on top of the structural analysis for [context-aware validation](https://codepathfinder.dev/blog/introducing-secureflow-cli-to-hunt-vuln).
- **IDE and CLI**: Works in [VS Code](https://codepathfinder.dev/docs/quickstart), from the command line, and in CI/CD pipelines.

## How it's different

- **Call graphs and dataflow**: Indexes [functions, endpoints, DB calls, and dataflows](https://codepathfinder.dev/blog/static-analysis-isnt-enough-understanding-library-interactions-for-effective-data-flow-tracking) to trace source-to-sink vulnerabilities instead of matching syntax patterns.
- **LLMs validate, don't detect**: The structural analysis finds potential issues; [LLMs explain and prioritize](https://github.blog/ai-and-ml/llms/how-ai-enhances-static-application-security-testing-sast/) them. This keeps results reproducible.
- **Your code stays local**: You [bring your own API keys](https://codepathfinder.dev/secureflow-ai) and talk directly to providers. No vendor-side code ingestion.

## Where to use it

- **AI coding assistants**: Run as an [MCP server](https://codepathfinder.dev/mcp) to give Claude Code, Cline, or other AI assistants deep code intelligence (call graphs, data flows, security patterns) beyond what LSP provides
- **In-editor**: SecureFlow VS Code extension ([VS Code Marketplace](https://marketplace.visualstudio.com/items?itemName=codepathfinder.secureflow) | [Open VSX](https://open-vsx.org/extension/codepathfinder/secureflow)) runs security checks as you type and catches vulnerable patterns before commit
- **Command line**: [SecureFlow CLI](https://www.npmjs.com/package/@codepathfinder/secureflow-cli) runs agentic loops over your repo to profile, read, trace, and validate vulnerabilities
- **CI/CD pipelines**: Exports to SARIF for [GitHub Advanced Security](https://github.com/shivasurya/code-pathfinder), integrates with DefectDojo, and supports custom rules for automated security gates

## Tools and workflows

**[Code Pathfinder CLI](https://codepathfinder.dev/blog/codeql-oss-alternative)**
The core scanner and query engine. Run it three ways: `scan` mode for security analysis with custom rules, `serve` mode as an [MCP server](https://codepathfinder.dev/mcp) for AI coding assistants (Claude Code, Cline), or `ci` mode in GitHub Actions and CI/CD pipelines. Indexes your codebase into call graphs and data flows, then runs Python-based security rules to find source-to-sink vulnerabilities.

**[SecureFlow CLI](https://www.npmjs.com/package/@codepathfinder/secureflow-cli)**
AI-powered vulnerability scanner that runs multi-turn analysis loops. First profiles your project to detect the stack (Django, Flask, FastAPI, etc.), then iteratively requests relevant files, traces data flows, and uses LLMs (Claude, GPT, Gemini, Grok, Ollama) to identify and explain security issues. Exports findings to JSON, SARIF, or DefectDojo format.

**SecureFlow VS Code extension** ([Marketplace](https://marketplace.visualstudio.com/items?itemName=codepathfinder.secureflow) | [Open VSX](https://open-vsx.org/extension/codepathfinder/secureflow))
In-editor security analysis. Right-click to scan files or profiles, review findings in a sidebar with severity levels, file locations, and fix recommendations. Uses the same AI models as SecureFlow CLI. Catches SQL injection, XSS, deserialization bugs, and other OWASP Top 10 issues as you code.

**[Custom Rules](https://codepathfinder.dev/docs/rules)**
Write security rules in Python using the PathFinder SDK. Query the code graph with `find_symbol()`, trace calls with `get_callees()` and `get_callers()`, check for vulnerable patterns. Rules run during `scan` or `ci` commands. See [rule registry](https://codepathfinder.dev/registry) for 50+ examples (SQL injection, RCE, privilege escalation, container misconfigurations).

## Supported Languages

- **[Python](https://codepathfinder.dev/registry/python)**: Full support for security analysis and vulnerability detection
- **[Docker](https://codepathfinder.dev/registry/docker)**: Dockerfile security scanning
- **[Docker Compose](https://codepathfinder.dev/registry/docker-compose)**: Configuration analysis and security checks
- **Go**: Coming soon

## Installation

### Homebrew (Recommended)

The easiest way to install on macOS or Linux. Available from version 0.0.34 onwards.

```bash
brew install shivasurya/tap/pathfinder
```

### pip

Install via pip to get both the CLI binary and Python SDK for writing security rules.

```bash
pip install codepathfinder
```

**Verify installation:**

```bash
# Test CLI binary
pathfinder --version

# Test Python SDK
python -c "from codepathfinder import rule, calls; print('SDK OK')"
```

**Supported platforms:** Linux (x86_64, aarch64), macOS (Intel, Apple Silicon), Windows (x64)

> **Migrating from npm?** The npm package is deprecated. Run `npm uninstall -g codepathfinder` then `pip install codepathfinder`.

### Docker

Ideal for CI/CD pipelines and containerized workflows.

```bash
docker pull shivasurya/code-pathfinder:stable-latest

# Run a scan
docker run --rm -v "./src:/src" \
  shivasurya/code-pathfinder:stable-latest \
  scan --project /src --rules /src/rules
```

### Pre-Built Binaries

Download platform-specific binaries from [GitHub Releases](https://github.com/shivasurya/code-pathfinder/releases). Available for Linux (amd64, arm64), macOS (Intel, Apple Silicon), and Windows (x64).

```bash
chmod u+x pathfinder
./pathfinder --help
```

### From Source

Build from source for the latest features. Requires Gradle and Go.

```bash
git clone https://github.com/shivasurya/code-pathfinder
cd code-pathfinder/sast-engine
gradle buildGo
./build/go/pathfinder --help
```


## Usage

### Scan Command (Interactive)

```bash
# Basic scan (text output to console)
pathfinder scan --rules rules/ --project /path/to/project

# With verbose output
pathfinder scan --rules rules/ --project . --verbose

# With debug output
pathfinder scan --rules rules/ --project . --debug

# JSON output to file
pathfinder scan --rules rules/ --project . --output json --output-file results.json

# SARIF output to file (GitHub Code Scanning compatible)
pathfinder scan --rules rules/ --project . --output sarif --output-file results.sarif

# CSV output to file
pathfinder scan --rules rules/ --project . --output csv --output-file results.csv

# JSON output to stdout (for piping)
pathfinder scan --rules rules/ --project . --output json | jq .

# Fail on specific severities
pathfinder scan --rules rules/ --project . --fail-on=critical,high
```

## GitHub Action

Add security scanning to your CI/CD pipeline.

**Best Practice:** Pin to a specific version (e.g., `@v1.2.0`) instead of `@main` to avoid breaking changes.

```yaml
# .github/workflows/security-scan.yml
name: Security Scan

on: [push, pull_request]

permissions:
  security-events: write
  contents: read

jobs:
  scan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6

      # Scan with remote Python rulesets
      - name: Run Python Security Scan
        uses: shivasurya/code-pathfinder@v1.2.0
        with:
          ruleset: python/deserialization, python/django, python/flask
          fail-on: critical,high

      - name: Upload SARIF
        uses: github/codeql-action/upload-sarif@v4
        if: always()
        with:
          sarif_file: pathfinder-results.sarif
```

**Scan Dockerfiles:**
```yaml
      - name: Run Docker Security Scan
        uses: shivasurya/code-pathfinder@v1.2.0
        with:
          ruleset: docker/security, docker/best-practice
```

**Use local rules:**
```yaml
      - name: Run Custom Rules
        uses: shivasurya/code-pathfinder@v1.2.0
        with:
          rules: python-sdk/examples/owasp_top10.py
```

### Action Inputs

| Input | Description | Default |
|-------|-------------|---------|
| `rules` | Path to Python SDK rules file or directory | - |
| `ruleset` | Remote ruleset(s) to use (e.g., `python/deserialization, docker/security`). Supports bundles or individual rule IDs. | - |
| `project` | Path to source code to scan | `.` |
| `output` | Output format: `sarif`, `json`, `csv`, `text` | `sarif` |
| `output-file` | Output file path | `pathfinder-results.sarif` |
| `fail-on` | Fail on severities (e.g., `critical,high`) | - |
| `verbose` | Enable verbose output with progress and statistics | `false` |
| `debug` | Enable debug diagnostics with timestamps | `false` |
| `skip-tests` | Skip scanning test files (test_*.py, *_test.py, etc.) | `true` |
| `refresh-rules` | Force refresh of cached rulesets (bypasses cache) | `false` |
| `disable-metrics` | Disable anonymous usage metrics collection | `false` |
| `python-version` | Python version to use | `3.12` |

**Note:** Either `rules` or `ruleset` must be specified.

### Available Remote Rulesets

**Python:**
- `python/deserialization` - Unsafe pickle.loads RCE detection
- `python/django` - Django SQL injection patterns
- `python/flask` - Flask security misconfigurations

**Docker:**
- `docker/security` - Critical and high-severity security issues
- `docker/best-practice` - Dockerfile optimization and best practices
- `docker/performance` - Performance optimization for container images

## Acknowledgements
Code Pathfinder uses tree-sitter for all language parsers.

## License

Licensed under [AGPL-3.0](https://github.com/shivasurya/code-pathfinder/blob/main/LICENSE).
