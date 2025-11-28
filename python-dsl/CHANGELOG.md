# Changelog

All notable changes to the codepathfinder Python DSL will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.1.0] - 2025-11-27

### Added
- **Positional argument matching** in `calls()` matcher via `match_position` parameter
  - Support for simple positional matching: `calls("open", match_position={1: "w"})`
  - Support for tuple indexing: `calls("socket.bind", match_position={"0[0]": "0.0.0.0"})`
  - Support for list of values: `calls("yaml.load", match_position={1: ["Loader", "UnsafeLoader"]})`
- **Keyword argument matching** in `calls()` matcher via `match_name` parameter
  - Example: `calls("app.run", match_name={"debug": True})`
- **Wildcard support in argument values**
  - Pattern matching in arguments: `calls("chmod", match_position={1: "0o7*"})`
  - IP address wildcards: `calls("connect", match_position={"0[0]": "192.168.*"})`
- **Type hints** added to `matchers.py` for better IDE support and type checking
- New `ArgumentValue` type alias for clearer type definitions

### Changed
- Enhanced `CallMatcher` class with argument constraint support
- Improved documentation with comprehensive examples for new features
- Updated IR serialization to include `positionalArgs` and `keywordArgs` fields

### Fixed
- Critical bugs in argument matching logic (PR #390)
- Tuple indexing for nested argument structures (PR #389)

### Technical Details
- Automatic wildcard detection in argument values (independent of pattern wildcards)
- Constraint propagation from pattern wildcards to argument constraints
- `matchMode` field changed from `match_mode` (camelCase consistency)

## [1.0.0] - 2025-11-09

### Added
- Initial release of codepathfinder Python DSL
- Core matchers: `calls()` and `variable()`
- Rule definition system with `@rule` decorator
- Dataflow analysis with `flows()`
- Propagation presets and custom propagation rules
- Logic operators: `And`, `Or`, `Not`
- Configuration system for default propagation and scope
- JSON IR generation for Go executor integration
- Comprehensive test suite with pytest
- Type hints and mypy support
- Black and Ruff formatting/linting configuration

### Features
- **Matchers**
  - `calls()`: Match function/method calls with wildcard support
  - `variable()`: Match variable references with patterns

- **Dataflow Analysis**
  - Source-to-sink tracking
  - Configurable propagation rules
  - Phase 1 and Phase 2 propagation presets

- **Rule System**
  - Decorator-based rule definitions
  - Severity levels (info, low, medium, high, critical)
  - Metadata support (CWE, OWASP references)

- **Logic Operators**
  - Combine matchers with And, Or, Not
  - Composable security patterns

### Documentation
- README with quickstart guide
- Inline examples and docstrings
- OWASP Top 10 example patterns

[1.1.0]: https://github.com/shivasurya/code-pathfinder/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/shivasurya/code-pathfinder/releases/tag/v1.0.0
