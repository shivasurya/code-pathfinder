"""
Decorators for Dockerfile and docker-compose rules.
"""

import atexit
import json
import sys
from typing import Callable, Dict, Any, List
from dataclasses import dataclass


@dataclass
class RuleMetadata:
    """Metadata for a container security rule."""

    id: str
    name: str = ""
    severity: str = "MEDIUM"
    category: str = "security"
    cwe: str = ""
    cve: str = ""
    tags: str = ""
    message: str = ""
    file_pattern: str = ""


@dataclass
class DockerfileRuleDefinition:
    """Complete definition of a Dockerfile rule."""

    metadata: RuleMetadata
    matcher: Dict[str, Any]
    rule_function: Callable


@dataclass
class ComposeRuleDefinition:
    """Complete definition of a docker-compose rule."""

    metadata: RuleMetadata
    matcher: Dict[str, Any]
    rule_function: Callable


# Global registries
_dockerfile_rules: List[DockerfileRuleDefinition] = []
_compose_rules: List[ComposeRuleDefinition] = []
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
        """Output all container rules as JSON when script ends."""
        if not _dockerfile_rules and not _compose_rules:
            return

        # Compile rules to JSON IR format
        from . import container_ir

        compiled = container_ir.compile_all_rules()

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


def dockerfile_rule(
    id: str,
    name: str = "",
    severity: str = "MEDIUM",
    category: str = "security",
    cwe: str = "",
    cve: str = "",
    tags: str = "",
    message: str = "",
) -> Callable:
    """
    Decorator for Dockerfile security rules.

    Example:
        @dockerfile_rule(id="DOCKER-001", severity="HIGH", cwe="CWE-250",
                        tags="security,docker,privilege-escalation")
        def container_runs_as_root():
            return missing(instruction="USER")
    """

    def decorator(func: Callable) -> Callable:
        # Get matcher from function
        matcher_result = func()

        # Convert to dict if it's a Matcher object
        if hasattr(matcher_result, "to_dict"):
            matcher_dict = matcher_result.to_dict()
        elif isinstance(matcher_result, dict):
            matcher_dict = matcher_result
        else:
            raise ValueError(f"Rule {id} must return a matcher or dict")

        # Create rule definition
        metadata = RuleMetadata(
            id=id,
            name=name or func.__name__.replace("_", " ").title(),
            severity=severity,
            category=category,
            cwe=cwe,
            cve=cve,
            tags=tags,
            message=message or f"Security issue detected by {id}",
            file_pattern="Dockerfile*",
        )

        rule_def = DockerfileRuleDefinition(
            metadata=metadata,
            matcher=matcher_dict,
            rule_function=func,
        )

        _dockerfile_rules.append(rule_def)
        _register_rule()  # Enable auto-execution if running as script

        # Return original function (can be called for testing)
        return func

    return decorator


def compose_rule(
    id: str,
    name: str = "",
    severity: str = "MEDIUM",
    category: str = "security",
    cwe: str = "",
    cve: str = "",
    tags: str = "",
    message: str = "",
) -> Callable:
    """
    Decorator for docker-compose security rules.

    Example:
        @compose_rule(id="COMPOSE-001", severity="HIGH", cwe="CWE-250",
                     tags="security,docker-compose,privilege-escalation")
        def privileged_service():
            return service_has(key="privileged", equals=True)
    """

    def decorator(func: Callable) -> Callable:
        matcher_result = func()

        if hasattr(matcher_result, "to_dict"):
            matcher_dict = matcher_result.to_dict()
        elif isinstance(matcher_result, dict):
            matcher_dict = matcher_result
        else:
            raise ValueError(f"Rule {id} must return a matcher or dict")

        metadata = RuleMetadata(
            id=id,
            name=name or func.__name__.replace("_", " ").title(),
            severity=severity,
            category=category,
            cwe=cwe,
            cve=cve,
            tags=tags,
            message=message or f"Security issue detected by {id}",
            file_pattern="**/docker-compose*.yml",
        )

        rule_def = ComposeRuleDefinition(
            metadata=metadata,
            matcher=matcher_dict,
            rule_function=func,
        )

        _compose_rules.append(rule_def)
        _register_rule()  # Enable auto-execution if running as script

        return func

    return decorator


def get_dockerfile_rules() -> List[DockerfileRuleDefinition]:
    """Get all registered Dockerfile rules."""
    return _dockerfile_rules.copy()


def get_compose_rules() -> List[ComposeRuleDefinition]:
    """Get all registered docker-compose rules."""
    return _compose_rules.copy()


def clear_rules():
    """Clear all registered rules (for testing)."""
    global _dockerfile_rules, _compose_rules
    _dockerfile_rules = []
    _compose_rules = []
