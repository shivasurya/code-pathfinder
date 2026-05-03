"""
Decorators for C security rules.

Mirrors `go_decorators.py` exactly. The only behavioural difference is the
language tag injected into dataflow IR: ``language="c"`` so the executor
scopes analysis to nodes with ``Node.Language == "c"``.

Pure ``calls()`` matchers (``type == "call_matcher"``) are NOT language-scoped,
matching the @go_rule contract — see PR-11 spec, Gap 1 / Gap 4.
"""

import atexit
import json
import sys
from typing import Callable, List
from dataclasses import dataclass


@dataclass
class CRuleMetadata:
    """Metadata for a C security rule."""

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
class CRuleDefinition:
    """Complete definition of a C security rule."""

    metadata: CRuleMetadata
    matcher: dict
    rule_function: Callable


_c_rules: List[CRuleDefinition] = []
_auto_execute_enabled = False


def _enable_auto_execute() -> None:
    """Enable automatic rule compilation and stdout JSON output at script exit."""
    global _auto_execute_enabled
    if _auto_execute_enabled:
        return
    _auto_execute_enabled = True

    def _output_rules():
        if not _c_rules:
            return
        from . import c_ir

        compiled = c_ir.compile_all_rules()
        print(json.dumps(compiled))

    atexit.register(_output_rules)


def _register_rule() -> None:
    """Enable auto-execute when a rule file is run as ``__main__``."""
    frame = sys._getframe(2)
    if frame.f_globals.get("__name__") == "__main__":
        _enable_auto_execute()


def c_rule(
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
    Decorator for C security rules. Mirrors @go_rule.

    Sets ``language="c"`` on the DataflowMatcher dict so DataflowExecutor
    scopes analysis to C functions only. Only affects flows() rules
    (``type=="dataflow"``); pure calls() rules remain language-agnostic.
    """

    def decorator(func: Callable) -> Callable:
        matcher_result = func()

        if hasattr(matcher_result, "to_ir"):
            matcher_dict = matcher_result.to_ir()
        elif hasattr(matcher_result, "to_dict"):
            matcher_dict = matcher_result.to_dict()
        elif isinstance(matcher_result, dict):
            matcher_dict = matcher_result
        else:
            raise ValueError(f"Rule {id} must return a matcher or dict")

        if isinstance(matcher_dict, dict) and matcher_dict.get("type") == "dataflow":
            matcher_dict["language"] = "c"

        metadata = CRuleMetadata(
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
        rule_def = CRuleDefinition(
            metadata=metadata,
            matcher=matcher_dict,
            rule_function=func,
        )
        _c_rules.append(rule_def)
        _register_rule()

        return func

    return decorator


def get_c_rules() -> List[CRuleDefinition]:
    """Return a snapshot of registered C rules."""
    return _c_rules.copy()


def clear_c_rules() -> None:
    """Clear all registered C rules (test isolation)."""
    global _c_rules
    _c_rules = []
