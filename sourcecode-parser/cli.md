# CLI Reference

## Commands

### scan

Interactive security scanning with human-readable output.

**Usage**:
```bash
pathfinder scan --rules <path> --project <path> [flags]
```

**Required Flags**:
- `--rules, -r` - Path to rules file or directory
- `--project, -p` - Path to project to scan

**Optional Flags**:
- `--verbose, -v` - Show progress and statistics
- `--debug` - Show debug diagnostics with timestamps
- `--fail-on` - Fail with exit code 1 if findings match severities

**Examples**:
```bash
# Basic scan
pathfinder scan --rules rules/security.py --project /app

# Verbose scan
pathfinder scan -r rules/ -p . -v

# CI-style failure
pathfinder scan -r rules/ -p . --fail-on=critical,high
```

---

### ci

CI/CD optimized scanning with machine-readable output.

**Usage**:
```bash
pathfinder ci --rules <path> --project <path> --output <format> [flags]
```

**Required Flags**:
- `--rules, -r` - Path to rules file or directory
- `--project, -p` - Path to project to scan
- `--output, -o` - Output format: json, csv, sarif

**Optional Flags**:
- `--verbose, -v` - Show progress and statistics (to stderr)
- `--debug` - Show debug diagnostics
- `--fail-on` - Fail with exit code 1 if findings match severities

**Examples**:
```bash
# JSON output
pathfinder ci -r rules/ -p . -o json > results.json

# SARIF for GitHub
pathfinder ci -r rules/ -p . -o sarif > results.sarif

# CSV with failure control
pathfinder ci -r rules/ -p . -o csv --fail-on=critical > results.csv
```

---

### diagnose

Diagnostic mode for debugging rule behavior.

**Usage**:
```bash
pathfinder diagnose --rules <path> --project <path> [flags]
```

---

### version

Display version information.

**Usage**:
```bash
pathfinder version
```

---

## Output Format Reference

### JSON Schema

| Field | Type | Description |
|-------|------|-------------|
| `tool.name` | string | "Code Pathfinder" |
| `tool.version` | string | Tool version |
| `scan.target` | string | Project path |
| `scan.rules_executed` | int | Number of rules |
| `results[]` | array | Detection results |
| `results[].rule_id` | string | Rule identifier |
| `results[].severity` | string | critical/high/medium/low |
| `results[].location.file` | string | File path |
| `results[].location.line` | int | Line number |
| `results[].detection.type` | string | pattern/taint-local/taint-global |
| `summary.total` | int | Total findings |
| `summary.by_severity` | object | Count by severity |

### CSV Columns

1. severity
2. confidence
3. rule_id
4. rule_name
5. cwe
6. owasp
7. file
8. line
9. column
10. function
11. message
12. detection_type
13. detection_scope
14. source_line
15. sink_line
16. tainted_var
17. sink_call

### SARIF 2.1.0

Compliant with [SARIF 2.1.0 specification](https://docs.oasis-open.org/sarif/sarif/v2.1.0/sarif-v2.1.0.html).

Features:
- Rule metadata with help text
- Code flows for taint analysis
- Related locations for sources
- Security severity scores
- URI base ID for portable paths

---

## Exit Code Reference

| Code | Constant | Description |
|------|----------|-------------|
| 0 | ExitCodeSuccess | No findings, or findings without --fail-on match |
| 1 | ExitCodeFindings | Findings match at least one --fail-on severity |
| 2 | ExitCodeError | Configuration or execution error |

### --fail-on Syntax

```bash
--fail-on=<severity>[,<severity>...]
```

Valid severities: `critical`, `high`, `medium`, `low`, `info`

Examples:
- `--fail-on=critical` - Fail only on critical
- `--fail-on=critical,high` - Fail on critical or high
- `--fail-on=critical,high,medium,low` - Fail on any finding
