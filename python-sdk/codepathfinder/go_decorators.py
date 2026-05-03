"""
Decorators for Go security rules.
Mirrors python_decorators.py for Go-specific rules.
"""

import atexit
import json
import sys
from typing import Callable, List
from dataclasses import dataclass


@dataclass
class GoRuleMetadata:
    """Metadata for a Go security rule."""

    id: str
    name: str = ""
    severity: str = "MEDIUM"
    category: str = "security"
    cwe: str = ""
    cve: str = ""
    tags: str = ""
    message: str = ""
    owasp: str = ""


@dataclass
class GoRuleDefinition:
    """Complete definition of a Go security rule."""

    metadata: GoRuleMetadata
    matcher: dict
    rule_function: Callable


_go_rules: List[GoRuleDefinition] = []
_auto_execute_enabled = False


def _enable_auto_execute() -> None:
    """Enable automatic rule compilation and output when script ends."""
    global _auto_execute_enabled
    if _auto_execute_enabled:
        return
    _auto_execute_enabled = True

    def _output_rules():
        if not _go_rules:
            return
        from . import go_ir

        compiled = go_ir.compile_all_rules()
        print(json.dumps(compiled))

    atexit.register(_output_rules)


def _register_rule() -> None:
    """
    Check if auto-execution should be enabled when a rule is registered.
    Enables auto-execution if the module is being executed directly (not imported).
    """
    frame = sys._getframe(2)  # Get caller's frame (the module defining the rule)
    if frame.f_globals.get("__name__") == "__main__":
        _enable_auto_execute()


def go_rule(
    id: str,
    name: str = "",
    severity: str = "MEDIUM",
    category: str = "security",
    cwe: str = "",
    cve: str = "",
    tags: str = "",
    message: str = "",
    owasp: str = "",
) -> Callable:
    """
    Decorator for Go security rules. Mirrors @python_rule.

    Sets language="go" on the DataflowMatcher dict so DataflowExecutor
    scopes analysis to Go functions only (via DataflowIR.Language from PR-06).
    """

    def decorator(func: Callable) -> Callable:
        matcher_result = func()

        # Convert matcher to IR dict
        if hasattr(matcher_result, "to_ir"):
            matcher_dict = matcher_result.to_ir()
        elif hasattr(matcher_result, "to_dict"):
            matcher_dict = matcher_result.to_dict()
        elif isinstance(matcher_result, dict):
            matcher_dict = matcher_result
        else:
            raise ValueError(f"Rule {id} must return a matcher or dict")

        # Inject language="go" into the DataflowIR matcher dict
        if isinstance(matcher_dict, dict) and matcher_dict.get("type") == "dataflow":
            matcher_dict["language"] = "go"

        metadata = GoRuleMetadata(
            id=id,
            name=name or func.__name__.replace("_", " ").title(),
            severity=severity,
            category=category,
            cwe=cwe,
            cve=cve,
            tags=tags,
            message=message or f"Security issue detected by {id}",
            owasp=owasp,
        )
        rule_def = GoRuleDefinition(
            metadata=metadata,
            matcher=matcher_dict,
            rule_function=func,
        )
        _go_rules.append(rule_def)
        _register_rule()

        return func

    return decorator


def get_go_rules() -> List[GoRuleDefinition]:
    """Get all registered Go rules."""
    return _go_rules.copy()


def clear_go_rules():
    """Clear all registered Go rules (for testing)."""
    global _go_rules
    _go_rules = []
