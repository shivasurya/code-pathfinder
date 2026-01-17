"""
JSON IR (Intermediate Representation) compiler for Python security rules.
"""

import json
from typing import List, Dict, Any

from .python_decorators import get_python_rules


def compile_python_rules() -> List[Dict[str, Any]]:
    """
    Compile all Python rules to JSON IR.

    Returns list of rule definitions ready for Go executor.
    """
    rules = get_python_rules()
    compiled = []

    for rule in rules:
        ir = {
            "id": rule.metadata.id,
            "name": rule.metadata.name,
            "severity": rule.metadata.severity,
            "category": rule.metadata.category,
            "cwe": rule.metadata.cwe,
            "cve": rule.metadata.cve,
            "tags": rule.metadata.tags,
            "message": rule.metadata.message,
            "owasp": rule.metadata.owasp,
            "rule_type": "python",
            "matcher": rule.matcher,
        }
        compiled.append(ir)

    return compiled


def compile_all_rules() -> Dict[str, List[Dict[str, Any]]]:
    """
    Compile all Python rules to JSON IR.

    Returns dict with 'python' rule list.
    """
    return {
        "python": compile_python_rules(),
    }


def compile_to_json(pretty: bool = True) -> str:
    """
    Compile all rules to JSON string.

    Args:
        pretty: If True, format with indentation.

    Returns:
        JSON string of all compiled rules.
    """
    compiled = compile_all_rules()
    if pretty:
        return json.dumps(compiled, indent=2)
    return json.dumps(compiled)


def write_ir_file(filepath: str, pretty: bool = True):
    """
    Write compiled rules to JSON file.

    Args:
        filepath: Output file path.
        pretty: If True, format with indentation.
    """
    json_str = compile_to_json(pretty=pretty)
    with open(filepath, "w") as f:
        f.write(json_str)
