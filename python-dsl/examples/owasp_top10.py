#!/usr/bin/env python3
"""
OWASP Top 10 Security Rules using codepathfinder Python DSL.

This file demonstrates real-world security vulnerability detection patterns
using the integrated Python DSL → JSON IR → Go executor pipeline.
"""

from codepathfinder import calls, flows, rule
from codepathfinder.presets import PropagationPresets


@rule(
    id="owasp-a03-sqli",
    severity="critical",
    cwe="CWE-89",
    owasp="A03:2021",
)
def detect_sql_injection():
    """Detects SQL injection vulnerabilities where user input flows to SQL execution."""
    return flows(
        from_sources=[
            calls("request.GET"),
            calls("request.POST"),
            calls("request.args.get"),
            calls("request.form.get"),
            calls("input"),
        ],
        to_sinks=[
            calls("execute"),
            calls("executemany"),
            calls("*.execute"),
            calls("*.executemany"),
            calls("raw"),
        ],
        sanitized_by=[
            calls("escape"),
            calls("escape_string"),
            calls("*.escape"),
            calls("parameterize"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )


@rule(
    id="owasp-a03-cmdi",
    severity="critical",
    cwe="CWE-78",
    owasp="A03:2021",
)
def detect_command_injection():
    """Detects OS command injection where user input flows to system calls."""
    return flows(
        from_sources=[
            calls("request.*"),
            calls("input"),
            calls("*.GET"),
            calls("*.POST"),
        ],
        to_sinks=[
            calls("system"),
            calls("popen"),
            calls("exec"),
            calls("os.system"),
            calls("subprocess.*"),
            calls("Runtime.exec"),
        ],
        sanitized_by=[
            calls("shlex.quote"),
            calls("pipes.quote"),
            calls("*.sanitize"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )


@rule(
    id="owasp-a03-codeinj",
    severity="critical",
    cwe="CWE-94",
    owasp="A03:2021",
)
def detect_code_injection():
    """Detects code injection where user input flows to eval/exec."""
    return flows(
        from_sources=[
            calls("request.*"),
            calls("input"),
            calls("*.GET"),
            calls("*.POST"),
        ],
        to_sinks=[
            calls("eval"),
            calls("exec"),
            calls("compile"),
            calls("execfile"),
            calls("__import__"),
        ],
        sanitized_by=[
            calls("ast.literal_eval"),
            calls("json.loads"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )


@rule(
    id="owasp-a10-ssrf",
    severity="high",
    cwe="CWE-918",
    owasp="A10:2021",
)
def detect_ssrf():
    """Detects SSRF where user input flows to HTTP requests."""
    return flows(
        from_sources=[
            calls("request.*"),
            calls("*.GET"),
            calls("*.POST"),
        ],
        to_sinks=[
            calls("requests.get"),
            calls("requests.post"),
            calls("urllib.request.urlopen"),
            calls("*.fetch"),
            calls("*.download"),
        ],
        sanitized_by=[
            calls("validate_url"),
            calls("is_safe_url"),
            calls("*.whitelist_check"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )


@rule(
    id="owasp-a01-path-traversal",
    severity="high",
    cwe="CWE-22",
    owasp="A01:2021",
)
def detect_path_traversal():
    """Detects path traversal where user input flows to file operations."""
    return flows(
        from_sources=[
            calls("request.*"),
            calls("*.GET"),
            calls("*.POST"),
            calls("input"),
        ],
        to_sinks=[
            calls("open"),
            calls("*.open"),
            calls("*.read"),
            calls("*.write"),
            calls("os.path.join"),
        ],
        sanitized_by=[
            calls("os.path.basename"),
            calls("pathlib.Path.resolve"),
            calls("*.safe_join"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )


@rule(
    id="owasp-a08-deserial",
    severity="high",
    cwe="CWE-502",
    owasp="A08:2021",
)
def detect_insecure_deserialization():
    """Detects insecure deserialization where user input flows to pickle/yaml."""
    return flows(
        from_sources=[
            calls("request.*"),
            calls("*.GET"),
            calls("*.POST"),
        ],
        to_sinks=[
            calls("pickle.loads"),
            calls("pickle.load"),
            calls("yaml.load"),
            calls("*.deserialize"),
        ],
        sanitized_by=[
            calls("yaml.safe_load"),
            calls("json.loads"),
        ],
        propagates_through=PropagationPresets.minimal(),
        scope="local",
    )


if __name__ == "__main__":
    import json

    # Execute all rules and output JSON IR
    rules = [
        detect_sql_injection.execute(),
        detect_command_injection.execute(),
        detect_code_injection.execute(),
        detect_ssrf.execute(),
        detect_path_traversal.execute(),
        detect_insecure_deserialization.execute(),
    ]

    print(json.dumps(rules, indent=2))
