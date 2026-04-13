"""
JSON IR (Intermediate Representation) compiler for Go security rules.
Mirrors python_ir.py.
"""

from typing import List, Dict, Any

from .go_decorators import get_go_rules


def compile_go_rules() -> List[Dict[str, Any]]:
    """
    Compile all Go rules to JSON IR format expected by Go executor.

    Emits "language": "go" in rule metadata for display/filtering.
    The language field is ALSO inside the matcher dict (set by @go_rule
    decorator) for DataflowExecutor runtime filtering.
    """
    rules = get_go_rules()
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
                "language": "go",
            },
            "matcher": rule.matcher,
        }
        compiled.append(ir)

    return compiled


def compile_all_rules() -> List[Dict[str, Any]]:
    """Compile all Go rules to JSON IR array format."""
    return compile_go_rules()
