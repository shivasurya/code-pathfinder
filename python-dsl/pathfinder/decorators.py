"""
Decorators for pathfinder rule definitions.

The @rule decorator marks functions as security patterns.
"""

from typing import Callable, Optional
from .ir import serialize_ir


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
        return Rule(id=id, severity=severity, func=func, cwe=cwe, owasp=owasp)

    return decorator
