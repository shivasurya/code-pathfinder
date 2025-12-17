"""
Decorators for pathfinder rule definitions.

The @rule decorator marks functions as security patterns.
"""

import atexit
import json
import sys
from typing import Callable, Optional, List
from .ir import serialize_ir


# Global registry for auto-execution
_rule_registry: List["Rule"] = []
_auto_execute_enabled = False


def _enable_auto_execute() -> None:
    """
    Enable automatic rule execution when script ends.

    This should be called once when the first rule is registered
    and the module is being executed as a script (not imported).
    """
    global _auto_execute_enabled
    if _auto_execute_enabled:
        return

    _auto_execute_enabled = True

    def _output_rules():
        """Output all registered rules as JSON when script ends."""
        if not _rule_registry:
            return

        # Execute all rules and collect their JSON IR
        rules_json = [rule.execute() for rule in _rule_registry]

        # Output to stdout for Go loader to capture
        print(json.dumps(rules_json))

    # Register cleanup handler
    atexit.register(_output_rules)


def _register_rule(rule_obj: "Rule") -> None:
    """
    Register a rule for auto-execution.

    Args:
        rule_obj: The Rule instance to register
    """
    _rule_registry.append(rule_obj)

    # Enable auto-execution on first rule registration
    # Check if module is being executed directly (not imported)
    frame = sys._getframe(2)  # Get caller's frame (the module defining the rule)
    if frame.f_globals.get("__name__") == "__main__":
        _enable_auto_execute()


class Rule:
    """
    Represents a security rule with metadata.

    Attributes:
        id: Unique rule identifier (e.g., "sqli-001")
        name: Human-readable name (defaults to function name)
        severity: critical | high | medium | low
        cwe: CWE identifier (e.g., "CWE-89")
        owasp: OWASP category (e.g., "A03:2021")
        description: What this rule detects (from docstring)
        matcher: The matcher/combinator returned by the rule function
    """

    def __init__(
        self,
        id: str,
        severity: str,
        func: Callable,
        cwe: Optional[str] = None,
        owasp: Optional[str] = None,
    ):
        self.id = id
        self.name = func.__name__
        self.severity = severity
        self.cwe = cwe
        self.owasp = owasp
        self.description = func.__doc__ or ""
        self.func = func

    def execute(self) -> dict:
        """
        Execute the rule function and serialize to JSON IR.

        Returns:
            {
                "rule": {
                    "id": "sqli-001",
                    "name": "detect_sql_injection",
                    "severity": "critical",
                    "cwe": "CWE-89",
                    "owasp": "A03:2021",
                    "description": "Detects SQL injection vulnerabilities"
                },
                "matcher": {
                    "type": "call_matcher",
                    "patterns": ["execute"],
                    "wildcard": false
                }
            }
        """
        matcher = self.func()
        return {
            "rule": {
                "id": self.id,
                "name": self.name,
                "severity": self.severity,
                "cwe": self.cwe,
                "owasp": self.owasp,
                "description": self.description.strip(),
            },
            "matcher": serialize_ir(matcher),
        }


def rule(
    id: str,
    severity: str,
    cwe: Optional[str] = None,
    owasp: Optional[str] = None,
) -> Callable[[Callable], Rule]:
    """
    Decorator to mark a function as a security rule.

    Args:
        id: Unique rule identifier
        severity: critical | high | medium | low
        cwe: Optional CWE identifier
        owasp: Optional OWASP category

    Returns:
        Decorator function

    Example:
        @rule(id="code-injection", severity="critical", cwe="CWE-94")
        def detect_code_injection():
            '''Detects code injection via eval'''
            return calls("eval", "exec")
    """

    def decorator(func: Callable) -> Rule:
        rule_obj = Rule(id=id, severity=severity, func=func, cwe=cwe, owasp=owasp)
        _register_rule(rule_obj)
        return rule_obj

    return decorator
