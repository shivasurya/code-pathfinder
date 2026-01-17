"""
Decorators for Python security rules.
"""

import atexit
import json
import sys
from typing import Callable, Dict, Any, List
from dataclasses import dataclass


@dataclass
class PythonRuleMetadata:
    """Metadata for a Python security rule."""

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
class PythonRuleDefinition:
    """Complete definition of a Python security rule."""

    metadata: PythonRuleMetadata
    matcher: Dict[str, Any]
    rule_function: Callable


# Global registry
_python_rules: List[PythonRuleDefinition] = []
_auto_execute_enabled = False


def _enable_auto_execute() -> None:
    """
    Enable automatic rule compilation and output when script ends.

    This provides consistent behavior with code analysis rules -
    no __main__ block needed.
    """
    global _auto_execute_enabled
    if _auto_execute_enabled:
        return

    _auto_execute_enabled = True

    def _output_rules():
        """Output all Python rules as JSON when script ends."""
        if not _python_rules:
            return

        # Compile rules to JSON IR format
        from . import python_ir

        compiled = python_ir.compile_all_rules()

        # Output to stdout for Go loader to capture
        print(json.dumps(compiled))

    # Register cleanup handler
    atexit.register(_output_rules)


def _register_rule() -> None:
    """
    Check if auto-execution should be enabled when a rule is registered.

    Enables auto-execution if the module is being executed directly (not imported).
    """
    # Check if module is being executed directly
    frame = sys._getframe(2)  # Get caller's frame (the module defining the rule)
    if frame.f_globals.get("__name__") == "__main__":
        _enable_auto_execute()


def python_rule(
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
    Decorator for Python security rules.

    Example:
        @python_rule(
            id="PYTHON-001",
            severity="CRITICAL",
            cwe="CWE-89",
            owasp="A03:2021",
            tags="python,sql-injection,django,database"
        )
        def detect_sql_injection():
            return flows(
                from_sources=[calls("request.GET")],
                to_sinks=[calls("cursor.execute")],
                scope="local"
            )

    Args:
        id: Unique rule identifier (e.g., "PYTHON-DJANGO-001")
        name: Human-readable rule name (auto-generated from function name if not provided)
        severity: Rule severity (CRITICAL, HIGH, MEDIUM, LOW, INFO)
        category: Rule category (security, django, flask, deserialization, etc.)
        cwe: CWE identifier (e.g., "CWE-89")
        cve: CVE identifier (e.g., "CVE-2022-34265")
        tags: Comma-separated tags (e.g., "python,django,sql-injection")
        message: Detection message
        owasp: OWASP category (e.g., "A03:2021")

    Returns:
        Decorated function that registers the rule
    """

    def decorator(func: Callable) -> Callable:
        # Get matcher from function
        matcher_result = func()

        # Convert to dict if it's a Matcher object
        if hasattr(matcher_result, "to_ir"):
            matcher_dict = matcher_result.to_ir()
        elif hasattr(matcher_result, "to_dict"):
            matcher_dict = matcher_result.to_dict()
        elif isinstance(matcher_result, dict):
            matcher_dict = matcher_result
        else:
            raise ValueError(f"Rule {id} must return a matcher or dict")

        # Create rule definition
        metadata = PythonRuleMetadata(
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

        rule_def = PythonRuleDefinition(
            metadata=metadata,
            matcher=matcher_dict,
            rule_function=func,
        )

        _python_rules.append(rule_def)
        _register_rule()  # Enable auto-execution if running as script

        # Return original function (can be called for testing)
        return func

    return decorator


def get_python_rules() -> List[PythonRuleDefinition]:
    """Get all registered Python rules."""
    return _python_rules.copy()


def clear_rules():
    """Clear all registered rules (for testing)."""
    global _python_rules
    _python_rules = []
