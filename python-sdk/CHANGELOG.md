# Changelog

All notable changes to the codepathfinder Python SDK will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.0.1] - 2026-03-27

### Added
- `attribute()` matcher for taint sources that are property accesses, not function calls (e.g., `request.url`, `file.filename`, `request.data`)
- `QueryType.attr()` for type-constrained attribute matching (e.g., `FlaskRequest.attr("url", "host")`)
- `AttributeMethodMatcher` class returned by `QueryType.attr()`
- `ATTRIBUTE_MATCHER` and `TYPE_CONSTRAINED_ATTRIBUTE` IR types
- `attribute()` works inside `flows()` as source, sink, or sanitizer

### Changed
- `AnyMatcher` union type now includes `AttributeMatcher` and `AttributeMethodMatcher`

## [2.0.0] - 2026-03-21

### Added
- Inter-procedural taint analysis with `scope="global"` for cross-file dataflow tracking
- Source and sink file paths in JSON, SARIF, CSV, and text output for cross-file findings
- 190+ security rules across Python, Docker, and Docker Compose
- `PropagationPresets.standard()` for common taint propagation patterns
- QueryType system for type-aware detection replacing pattern-based matching
- `.tracks(N)` parameter targeting on sinks for precise argument-position filtering
- GitHub Actions deploy workflow for sandbox server

### Fixed
- Cross-file taint findings now report the correct sink file instead of the source file
- Fallback sink matchers now try when tracked params reject on type-constrained calls
- Dedup identical findings from multiple matcher paths in global analysis
- F-string and concat expression taint propagation in function arguments

### Changed
- Default dev version bumped from 1.2.2 to 2.0.0

## [1.3.2] - 2026-01-29

### Fixed
- Enable instance variable method call resolution (#495)
- Fix static method call resolution for cross-file calls (#494)

## [1.3.1] - 2026-01-28

### Documentation
- Improved README clarity and consistency across all components (#487)

## [1.3.0] - 2026-01-25

No python-sdk specific changes. Version bump for binary compatibility.

## [1.2.2] - 2026-01-23

### Changed
- Enhanced CLI output with progress bars and banner system (#473, #474, #476)
- Improved verbose logging for better user experience (#475)

### Fixed
- PyPI publish workflow restricted to release events only (#477)

## [1.2.1] - 2026-01-20

### Added
- Category-level ruleset expansion with `docker/all` syntax (#471)

### Changed
- **GitHub Action rewritten as composite action** using pip installation (#465)
  - Replaced Docker-based action with faster pip-based installation
  - Fixed incorrect `--ruleset` flag to proper `--rules` flag
  - Uses `scan` command instead of deprecated `ci` command
  - Added support for `fail-on`, `verbose`, `skip-tests` options
  - Added `python-version` input for flexibility
  - Outputs `results-file` and `version` for downstream steps

## [1.2.0] - 2026-01-18

### Added
- **Remote ruleset loading from codepathfinder.dev** (#466)
  - Load pre-built security rulesets: `--ruleset docker/security`
  - Python security rules for R2 CDN distribution (#469)

### Removed
- Obsolete deprecation-notice.js (#468)

## [1.1.6] - 2026-01-16

### Changed
- Updated Python SDK package description (#462)

## [1.1.5] - 2026-01-14

No python-sdk specific changes. Version bump for binary compatibility.

## [1.1.4] - 2026-01-12

### Added
- **CLI wrapper module for automatic binary management** (#442)
  - Downloads platform-specific pathfinder binary on first use
  - Manages binary installation in user's environment
  - Transparent binary execution from Python

### Changed
- **Migrated from npm to PyPI distribution** (#444)
  - Deprecated npm package in favor of `pip install codepathfinder`
  - Platform wheel build workflow for PyPI (#443)
  - Automatic binary downloads per platform

### Fixed
- Include rules package in distribution (#445)

## [1.1.3] - 2026-01-10

### Fixed
- PyPI workflow simplifications and condition checks (#446-450)

## [1.1.2] - 2026-01-08

### Added
- **JSON/SARIF/CSV output formats** with file output support (#432)
  - `--output-format json|sarif|csv|text`
  - `--output-file <path>` for saving results
- **Auto-execution support for Python SDK rules** (#435)
  - Rules execute automatically when scan completes
  - Streamlined workflow without manual execution

### Fixed
- `/lib64` bind mount to nsjail for Python SDK rule loading (#438)
- Removed hardcoded version in JSON/SARIF formatters (#436)

### Removed
- Playground directory and dependencies (#433)

## [1.1.1] - 2026-01-05

### Added
- **Docker container security rules** expanded from 18 to 47 rules (#428)
  - Container security rule executor and infrastructure (#422)
  - Python SDK advanced features for Docker rules (#421)
  - Python SDK core for container rules (#420)
  - Docker-compose parser with security queries (#419)
  - Comprehensive instruction converters for all Dockerfile instructions (#418)
  - Tree-sitter Dockerfile parsing integration (#417)
  - Core data structures for Dockerfile parsing (#416)

## [1.1.0] - 2025-11-27

### Added
- **Initial PyPI release of codepathfinder Python SDK**
- Python SDK for writing custom security rules
- Rule execution with dataflow analysis
- Multiple output formats: JSON, SARIF, CSV, text
- Binary distribution with automatic platform detection

### Features
- **Python SDK Rule System**
  - Decorator-based rule definitions with `@rule`
  - Matchers: `calls()` and `variable()` with wildcard support
  - Dataflow analysis: source-to-sink tracking
  - Logic operators: And, Or, Not for composable patterns
  - Severity levels (info, low, medium, high, critical)
  - Metadata support (CWE, OWASP references)

- **Multi-language Support**
  - Python code analysis
  - Java code analysis
  - Dockerfile security scanning
  - docker-compose.yml analysis

- **CLI Interface**
  - `pathfinder scan` - Scan project with local rules
  - `pathfinder ci` - CI/CD integration mode
  - Configurable output formats and destinations

[1.3.2]: https://github.com/shivasurya/code-pathfinder/compare/v1.3.1...v1.3.2
[1.3.1]: https://github.com/shivasurya/code-pathfinder/compare/v1.3.0...v1.3.1
[1.3.0]: https://github.com/shivasurya/code-pathfinder/compare/v1.2.2...v1.3.0
[1.2.2]: https://github.com/shivasurya/code-pathfinder/compare/v1.2.1...v1.2.2
[1.2.1]: https://github.com/shivasurya/code-pathfinder/compare/v1.2.0...v1.2.1
[1.2.0]: https://github.com/shivasurya/code-pathfinder/compare/v1.1.6...v1.2.0
[1.1.6]: https://github.com/shivasurya/code-pathfinder/compare/v1.1.5...v1.1.6
[1.1.5]: https://github.com/shivasurya/code-pathfinder/compare/v1.1.4...v1.1.5
[1.1.4]: https://github.com/shivasurya/code-pathfinder/compare/v1.1.3...v1.1.4
[1.1.3]: https://github.com/shivasurya/code-pathfinder/compare/v1.1.2...v1.1.3
[1.1.2]: https://github.com/shivasurya/code-pathfinder/compare/v1.1.1...v1.1.2
[1.1.1]: https://github.com/shivasurya/code-pathfinder/compare/v1.1.0...v1.1.1
[1.1.0]: https://github.com/shivasurya/code-pathfinder/releases/tag/v1.1.0
