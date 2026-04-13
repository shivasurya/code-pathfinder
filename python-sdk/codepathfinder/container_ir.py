"""
JSON IR (Intermediate Representation) compiler for container rules.
"""

import json
from typing import List, Dict, Any

from .container_decorators import (
    get_dockerfile_rules,
    get_compose_rules,
)


def compile_dockerfile_rules() -> List[Dict[str, Any]]:
    """
    Compile all Dockerfile rules to JSON IR.

    Returns list of rule definitions ready for Go executor.
    """
    rules = get_dockerfile_rules()
    compiled = []

    for rule in rules:
        ir = {
            "id": rule.metadata.id,
            "name": rule.metadata.name,
            "severity": rule.metadata.severity,
            "category": rule.metadata.category,
            "cwe": rule.metadata.cwe,
            "message": rule.metadata.message,
            "file_pattern": rule.metadata.file_pattern,
            "rule_type": "dockerfile",
            "matcher": rule.matcher,
        }
        compiled.append(ir)

    return compiled


def compile_compose_rules() -> List[Dict[str, Any]]:
    """
    Compile all docker-compose rules to JSON IR.

    Returns list of rule definitions ready for Go executor.
    """
    rules = get_compose_rules()
    compiled = []

    for rule in rules:
        ir = {
            "id": rule.metadata.id,
            "name": rule.metadata.name,
            "severity": rule.metadata.severity,
            "category": rule.metadata.category,
            "cwe": rule.metadata.cwe,
            "message": rule.metadata.message,
            "file_pattern": rule.metadata.file_pattern,
            "rule_type": "compose",
            "matcher": rule.matcher,
        }
        compiled.append(ir)

    return compiled


def compile_all_rules() -> Dict[str, List[Dict[str, Any]]]:
    """
    Compile all container rules to JSON IR.

    Returns dict with 'dockerfile' and 'compose' rule lists.
    """
    return {
        "dockerfile": compile_dockerfile_rules(),
        "compose": compile_compose_rules(),
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
