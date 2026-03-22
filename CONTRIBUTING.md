# Contributing to Code Pathfinder

Thanks for your interest in contributing. This guide covers everything you need to set up the project locally, run tests, and submit a pull request.

## Contributor License Agreement (CLA)

**All contributors must sign the CLA before any pull request can be merged.**

Read the full agreement: [CLA.md](./CLA.md)

By submitting a pull request, you confirm that you have read and agree to the CLA. You may also explicitly sign by posting this comment on your PR:

> "I have read the CLA Document and I hereby sign the CLA"

Corporate contributors should contact shiva@codepathfinder.dev before submitting.

## Project Structure

```
code-pathfinder/
├── sast-engine/          # Core analysis engine (Go)
│   ├── cmd/              # CLI commands (scan, serve, ci, version)
│   ├── dsl/              # Dataflow execution, IR types, matchers
│   ├── graph/            # AST parsing, call graph, taint analysis
│   │   └── callgraph/    # Call graph builder, VDG, taint propagation
│   ├── output/           # Formatters (JSON, SARIF, CSV, text)
│   ├── ruleset/          # CDN ruleset resolver
│   └── mcp/              # MCP server for AI assistants
├── python-sdk/           # Python SDK for writing security rules
├── rules/                # Security rule definitions (YAML + Python)
│   ├── python/           # Python rules (django, flask, lang, etc.)
│   ├── docker/           # Dockerfile rules
│   └── docker-compose/   # Docker Compose rules
├── extension/            # VS Code extension
├── scripts/              # Build and release scripts
└── tools/                # Rule processing, typeshed converter
```

## Prerequisites

- **Go 1.26.1+** — the engine is written in Go
- **Python 3.8+** — for the SDK and writing/testing rules
- **Gradle** — build system (optional, you can use `go` commands directly)
- **Git** — for version control

## Setting Up Locally

### 1. Fork and clone

```bash
# Fork on GitHub, then:
git clone https://github.com/YOUR-USERNAME/code-pathfinder.git
cd code-pathfinder
```

### 2. Build the engine

```bash
cd sast-engine
go mod download
go build -o build/go/pathfinder .
```

Verify:

```bash
./build/go/pathfinder version
```

### 3. Install the Python SDK

```bash
cd ../python-sdk
pip install -e .
python -c "from codepathfinder import rule; print('SDK OK')"
```

### 4. Run the engine against a test project

```bash
cd ../sast-engine
./build/go/pathfinder scan --ruleset python/all --project ../path/to/test/project
```

## Running Tests

### Go tests (engine)

```bash
cd sast-engine

# Run all tests
go test ./... -count=1

# Run a specific package
go test ./dsl/ -count=1

# Run a specific test
go test ./dsl/ -run TestTrackedParam -v

# Run with race detector
go test ./... -race -count=1
```

### Python SDK tests

```bash
cd python-sdk
pip install -e ".[dev]"
python -m pytest tests/ -v
```

### Typeshed converter tests

```bash
cd sast-engine/tools/typeshed-converter
python -m pytest test_convert.py test_mro.py -v
```

### Linting

```bash
cd sast-engine

# Install golangci-lint (https://golangci-lint.run/welcome/install/)
golangci-lint run
```

### Using Gradle (alternative)

```bash
# From project root
gradle buildGo      # Build the binary
gradle testGo       # Run Go tests
gradle lintGo       # Run linter
gradle clean         # Clean build artifacts
```

## What CI Checks

Every pull request runs these checks automatically. **All must pass before merge.**

| Check | What it does | How to run locally |
|-------|-------------|-------------------|
| **Go build** | Compiles the engine binary | `cd sast-engine && go build .` |
| **Go tests** | Unit + integration tests | `cd sast-engine && go test ./... -count=1` |
| **Go lint** | golangci-lint v2.11.3 | `cd sast-engine && golangci-lint run` |
| **Python SDK install** | Installs SDK from local source | `pip install -e ./python-sdk` |
| **Typeshed tests** | Converter tests with 90% coverage | `cd sast-engine/tools/typeshed-converter && pytest` |

Run all of these locally before pushing to avoid CI failures.

## Writing Security Rules

Rules live in `rules/<language>/<bundle>/<RULE-ID>/`. Each rule has:

```
RULE-ID/
├── meta.yaml      # Metadata: id, name, severity, CWE, description
├── rule.py         # Detection logic using the Python SDK
└── tests/
    ├── positive/   # Code that SHOULD trigger the rule
    └── negative/   # Code that should NOT trigger the rule
```

See the [rule writing guide](https://codepathfinder.dev/docs/rules) and existing rules in `rules/` for examples.

Test your rule locally:

```bash
# Run your rule against test fixtures
./sast-engine/build/go/pathfinder scan --rules ./rules/python/django/PYTHON-DJANGO-SEC-001/rule.py --project ./rules/python/django/PYTHON-DJANGO-SEC-001/tests/positive
```

## Submitting a Pull Request

### 1. Create a branch

```bash
git checkout -b your-branch-name
```

Use descriptive branch names: `fix/tracked-params-fallback`, `feat/go-language-support`, `rule/python-jwt-003`.

### 2. Make your changes

- Keep changes focused. One PR per feature or fix.
- Add tests for new functionality.
- Update existing tests if behavior changes.

### 3. Run tests locally

```bash
cd sast-engine
go test ./... -count=1
golangci-lint run
```

Fix any failures before pushing.

### 4. Commit and push

```bash
git add <specific-files>
git commit -m "fix: describe what you changed and why"
git push origin your-branch-name
```

Commit message format: `type: description`
- `fix:` — bug fix
- `feat:` — new feature
- `rule:` — new or updated security rule
- `docs:` — documentation changes
- `chore:` — maintenance, deps, CI

### 5. Open a pull request

- Open a PR against the `main` branch.
- Describe what you changed and why.
- Reference any related issues (e.g., "Fixes #123").
- Wait for CI checks to pass.
- Sign the CLA (see above).

### 6. Review process

A maintainer will review your PR. You may be asked to make changes. This is normal.

**Use PR comments to ask questions.** If you're unsure about an approach, ask before spending time on it. We'd rather help you get it right than have you redo work.

## Getting Help

- **GitHub Issues** — bug reports, feature requests: [github.com/shivasurya/code-pathfinder/issues](https://github.com/shivasurya/code-pathfinder/issues)
- **GitHub Discussions** — questions, ideas, general conversation: [github.com/shivasurya/code-pathfinder/discussions](https://github.com/shivasurya/code-pathfinder/discussions)
- **PR comments** — ask questions directly on your pull request
- **Email** — shiva@codepathfinder.dev

Don't hesitate to ask. No question is too basic.

## Code of Conduct

By participating in this project, you agree to abide by our [Code of Conduct](CODE_OF_CONDUCT.md).

## License

Code Pathfinder is licensed under [Apache-2.0](./LICENSE). By contributing, you agree that your contributions will be licensed under the same license, subject to the terms of the [CLA](./CLA.md).
