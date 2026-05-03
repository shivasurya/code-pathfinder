"""
JSON IR (Intermediate Representation) compiler for C security rules.

Mirrors `go_ir.py`. Emits ``language="c"`` in rule metadata for
display/filtering. The same field is also present inside the matcher dict
(injected by ``@c_rule``) for runtime DataflowExecutor scoping.
"""

from typing import List, Dict, Any

from .c_decorators import get_c_rules


def compile_c_rules() -> List[Dict[str, Any]]:
    """Compile all registered C rules into the JSON IR list expected by the Go executor."""
    rules = get_c_rules()
    compiled = []

    for rule in rules:
        ir = {
            "rule": {
                "id": rule.metadata.id,
                "name": rule.metadata.name,
                "severity": rule.metadata.severity.lower(),
                "cwe": rule.metadata.cwe,
                "owasp": rule.metadata.owasp,
                "description": rule.metadata.message
                or f"Security issue: {rule.metadata.id}",
                "language": "c",
            },
            "matcher": rule.matcher,
        }
        compiled.append(ir)

    return compiled


def compile_all_rules() -> List[Dict[str, Any]]:
    """Compile all C rules to the JSON IR array format."""
    return compile_c_rules()
