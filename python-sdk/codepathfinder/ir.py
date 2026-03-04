"""
JSON Intermediate Representation (IR) for pathfinder DSL.

The Python DSL serializes to JSON IR, which the Go executor consumes.
This enables language-agnostic pattern definitions (future: JS, Rust DSLs).
"""

from enum import Enum
from typing import Any, Dict, Protocol


class IRType(Enum):
    """IR node types for different matchers and combinators."""

    CALL_MATCHER = "call_matcher"
    VARIABLE_MATCHER = "variable_matcher"
    DATAFLOW = "dataflow"  # Coming in PR #3
    LOGIC_AND = "logic_and"  # Coming in PR #5
    LOGIC_OR = "logic_or"  # Coming in PR #5
    LOGIC_NOT = "logic_not"  # Coming in PR #5
    TYPE_CONSTRAINED_CALL = "type_constrained_call"
    RETURN_TYPE_CALL = "return_type_call"


class MatcherIR(Protocol):
    """Protocol for all matcher types (duck typing)."""

    def to_ir(self) -> Dict[str, Any]:
        """Serialize to JSON IR dictionary."""
        ...


def serialize_ir(matcher: MatcherIR) -> Dict[str, Any]:
    """
    Serialize any matcher to JSON IR.

    Args:
        matcher: Any object implementing MatcherIR protocol

    Returns:
        JSON-serializable dictionary

    Raises:
        AttributeError: If matcher doesn't implement to_ir()
    """
    if not hasattr(matcher, "to_ir"):
        raise AttributeError(f"{type(matcher).__name__} must implement to_ir() method")

    return matcher.to_ir()


def validate_ir(ir: Dict[str, Any]) -> bool:
    """
    Validate JSON IR structure.

    Args:
        ir: JSON IR dictionary

    Returns:
        True if valid, False otherwise

    Validates:
        - "type" field exists and is valid IRType
        - Required fields present for each type
    """
    if "type" not in ir:
        return False

    try:
        ir_type = IRType(ir["type"])
    except ValueError:
        return False

    # Type-specific validation
    if ir_type == IRType.CALL_MATCHER:
        return (
            "patterns" in ir
            and isinstance(ir["patterns"], list)
            and len(ir["patterns"]) > 0
            and "wildcard" in ir
            and isinstance(ir["wildcard"], bool)
        )

    if ir_type == IRType.VARIABLE_MATCHER:
        return (
            "pattern" in ir
            and isinstance(ir["pattern"], str)
            and len(ir["pattern"]) > 0
            and "wildcard" in ir
            and isinstance(ir["wildcard"], bool)
        )

    if ir_type == IRType.DATAFLOW:
        return (
            "sources" in ir
            and isinstance(ir["sources"], list)
            and len(ir["sources"]) > 0
            and "sinks" in ir
            and isinstance(ir["sinks"], list)
            and len(ir["sinks"]) > 0
            and "sanitizers" in ir
            and isinstance(ir["sanitizers"], list)
            and "propagation" in ir
            and isinstance(ir["propagation"], list)
            and "scope" in ir
            and ir["scope"] in ["local", "global"]
        )

    if ir_type == IRType.TYPE_CONSTRAINED_CALL:
        return (
            "receiverType" in ir
            and isinstance(ir["receiverType"], str)
            and len(ir["receiverType"]) > 0
            and "methodName" in ir
            and isinstance(ir["methodName"], str)
            and len(ir["methodName"]) > 0
            and "minConfidence" in ir
            and isinstance(ir["minConfidence"], (int, float))
            and 0.0 <= ir["minConfidence"] <= 1.0
            and "fallbackMode" in ir
            and ir["fallbackMode"] in ("name", "none", "warn")
        )

    if ir_type == IRType.RETURN_TYPE_CALL:
        return (
            "returnType" in ir
            and isinstance(ir["returnType"], str)
            and len(ir["returnType"]) > 0
            and "minConfidence" in ir
            and isinstance(ir["minConfidence"], (int, float))
            and 0.0 <= ir["minConfidence"] <= 1.0
        )

    return True
