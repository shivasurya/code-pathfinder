"""
Core matchers for the pathfinder Python DSL.

These matchers generate JSON IR for the Go executor.
"""

from typing import Dict, Optional, Union, List, Any
from .ir import IRType

ArgumentValue = Union[str, int, float, bool, List[Union[str, int, float, bool]]]


class CallMatcher:
    """
    Matches function/method calls with optional argument constraints.

    Examples:
        calls("eval")                    # Exact match
        calls("eval", "exec")            # Multiple patterns
        calls("request.*")               # Wildcard (any request.* call)
        calls("*.json")                  # Wildcard (any *.json call)
        calls("app.run", match_name={"debug": True})  # Keyword argument matching
        calls("open", match_position={1: "w"})  # Positional argument matching
        calls("socket.bind", match_position={"0[0]": "0.0.0.0"})  # Tuple indexing
        calls("connect", match_position={"0[0]": "192.168.*"})  # Wildcard + tuple
    """

    def __init__(
        self,
        *patterns: str,
        match_position: Optional[Dict[int, ArgumentValue]] = None,
        match_name: Optional[Dict[str, ArgumentValue]] = None,
    ):
        """
        Args:
            *patterns: Function names to match. Supports wildcards (*).
            match_position: Match positional arguments by index or tuple index.
                           Examples: {0: "value"}, {1: ["a", "b"]}, {"0[0]": "0.0.0.0"}
            match_name: Match named/keyword arguments {name: value}

        Position indexing:
            - Simple: {0: "value"} matches first argument
            - Tuple: {"0[0]": "value"} matches first element of first argument tuple
            - Wildcard: {"0[0]": "192.168.*"} matches with wildcard pattern

        Raises:
            ValueError: If no patterns provided or pattern is empty
        """
        if not patterns:
            raise ValueError("calls() requires at least one pattern")

        if any(not p or not isinstance(p, str) for p in patterns):
            raise ValueError("All patterns must be non-empty strings")

        self.patterns = list(patterns)
        self.wildcard = any("*" in p for p in patterns)
        self.match_position = match_position or {}
        self.match_name = match_name or {}

    def _make_constraint(self, value: ArgumentValue) -> Dict[str, Any]:
        """
        Create an argument constraint from a value.

        Automatically detects wildcard characters in string values.

        Args:
            value: The argument value or list of values

        Returns:
            Dictionary with 'value' and 'wildcard' keys
        """
        # Check if wildcard characters are present in string values
        # NOTE: Argument wildcard is independent of pattern wildcard (self.wildcard)
        # Pattern wildcard applies to function name matching (e.g., "*.bind")
        # Argument wildcard applies to argument value matching (e.g., "192.168.*")
        has_wildcard = False
        if isinstance(value, str) and ("*" in value or "?" in value):
            has_wildcard = True
        elif isinstance(value, list):
            has_wildcard = any(
                isinstance(v, str) and ("*" in v or "?" in v) for v in value
            )

        return {"value": value, "wildcard": has_wildcard}

    def to_ir(self) -> dict:
        """
        Serialize to JSON IR for Go executor.

        Returns:
            {
                "type": "call_matcher",
                "patterns": ["eval", "exec"],
                "wildcard": false,
                "matchMode": "any",
                "keywordArgs": { "debug": {"value": true, "wildcard": false} },
                "positionalArgs": { "0": {"value": "0.0.0.0", "wildcard": false} }
            }
        """
        ir = {
            "type": IRType.CALL_MATCHER.value,
            "patterns": self.patterns,
            "wildcard": self.wildcard,
            "matchMode": "any",
        }

        # Add positional argument constraints
        if self.match_position:
            positional_args = {}
            for pos, value in self.match_position.items():
                constraint = self._make_constraint(value)
                # Propagate wildcard flag from pattern to argument constraints
                if self.wildcard:
                    constraint["wildcard"] = True
                positional_args[str(pos)] = constraint
            ir["positionalArgs"] = positional_args

        # Add keyword argument constraints
        if self.match_name:
            keyword_args = {}
            for name, value in self.match_name.items():
                constraint = self._make_constraint(value)
                # Propagate wildcard flag from pattern to argument constraints
                if self.wildcard:
                    constraint["wildcard"] = True
                keyword_args[name] = constraint
            ir["keywordArgs"] = keyword_args

        return ir

    def __repr__(self) -> str:
        patterns_str = ", ".join(f'"{p}"' for p in self.patterns)
        return f"calls({patterns_str})"


class VariableMatcher:
    """
    Matches variable references by name.

    Examples:
        variable("user_input")           # Exact match
        variable("user_*")               # Wildcard prefix
        variable("*_id")                 # Wildcard suffix
    """

    def __init__(self, pattern: str):
        """
        Args:
            pattern: Variable name pattern. Supports wildcards (*).

        Raises:
            ValueError: If pattern is empty
        """
        if not pattern or not isinstance(pattern, str):
            raise ValueError("variable() requires a non-empty string pattern")

        self.pattern = pattern
        self.wildcard = "*" in pattern

    def to_ir(self) -> dict:
        """
        Serialize to JSON IR for Go executor.

        Returns:
            {
                "type": "variable_matcher",
                "pattern": "user_input",
                "wildcard": false
            }
        """
        return {
            "type": IRType.VARIABLE_MATCHER.value,
            "pattern": self.pattern,
            "wildcard": self.wildcard,
        }

    def __repr__(self) -> str:
        return f'variable("{self.pattern}")'


# Public API
def calls(
    *patterns: str,
    match_position: Optional[Dict[int, ArgumentValue]] = None,
    match_name: Optional[Dict[str, ArgumentValue]] = None,
) -> CallMatcher:
    """
    Create a matcher for function/method calls with optional argument constraints.

    Args:
        *patterns: Function names to match (supports wildcards)
        match_position: Match positional arguments by index {position: value}
        match_name: Match named/keyword arguments {name: value}

    Returns:
        CallMatcher instance

    Examples:
        >>> calls("eval")
        calls("eval")

        >>> calls("request.GET", "request.POST")
        calls("request.GET", "request.POST")

        >>> calls("urllib.*")
        calls("urllib.*")

        >>> calls("app.run", match_name={"debug": True})
        calls("app.run")

        >>> calls("socket.bind", match_position={0: "0.0.0.0"})
        calls("socket.bind")

        >>> calls("yaml.load", match_position={1: ["Loader", "UnsafeLoader"]})
        calls("yaml.load")

        >>> calls("chmod", match_position={1: "0o7*"})
        calls("chmod")

        >>> calls("app.run", match_position={0: "localhost"}, match_name={"debug": True})
        calls("app.run")
    """
    return CallMatcher(*patterns, match_position=match_position, match_name=match_name)


def variable(pattern: str) -> VariableMatcher:
    """
    Create a matcher for variable references.

    Args:
        pattern: Variable name pattern (supports wildcards)

    Returns:
        VariableMatcher instance

    Examples:
        >>> variable("user_input")
        variable("user_input")

        >>> variable("*_id")
        variable("*_id")
    """
    return VariableMatcher(pattern)


class TypeConstrainedCallMatcher:
    """
    Matches method calls on objects of a specific inferred type.

    Examples:
        calls_on("Cursor", "execute")              # Short name match
        calls_on("sqlite3.Cursor", "execute")       # Fully qualified
        calls_on("*Cursor", "execute")              # Wildcard prefix
        calls_on("sqlite3.*", "execute")             # Wildcard suffix
        calls_on("Cursor", "execute", fallback="none")  # No fallback
        calls_on("Cursor", "execute", min_confidence=0.8)  # Higher confidence
    """

    def __init__(
        self,
        receiver_type: str,
        method: str,
        min_confidence: float = 0.5,
        fallback: str = "name",
    ):
        """
        Args:
            receiver_type: Type name of the receiver object. Supports wildcards (*).
            method: Method name to match (e.g., "execute").
            min_confidence: Minimum type inference confidence (0.0-1.0).
            fallback: Behavior when type info is unavailable:
                - "name": match by method name only (default)
                - "none": skip (no match)
                - "warn": match but flag as low confidence

        Raises:
            ValueError: If receiver_type or method is empty, fallback is invalid,
                       or min_confidence is out of range.
        """
        if not receiver_type or not isinstance(receiver_type, str):
            raise ValueError("receiver_type must be a non-empty string")
        if not method or not isinstance(method, str):
            raise ValueError("method must be a non-empty string")
        if fallback not in ("name", "none", "warn"):
            raise ValueError(
                f"fallback must be 'name', 'none', or 'warn', got '{fallback}'"
            )
        if not isinstance(min_confidence, (int, float)) or not (
            0.0 <= min_confidence <= 1.0
        ):
            raise ValueError(
                f"min_confidence must be 0.0-1.0, got {min_confidence}"
            )

        self.receiver_type = receiver_type
        self.method = method
        self.min_confidence = float(min_confidence)
        self.fallback = fallback

    def to_ir(self) -> dict:
        """
        Serialize to JSON IR for Go executor.

        Returns:
            {
                "type": "type_constrained_call",
                "receiverType": "Cursor",
                "methodName": "execute",
                "minConfidence": 0.5,
                "fallbackMode": "name"
            }

        Note: JSON keys match Go struct tags in TypeConstrainedCallIR exactly.
        """
        return {
            "type": IRType.TYPE_CONSTRAINED_CALL.value,
            "receiverType": self.receiver_type,
            "methodName": self.method,
            "minConfidence": self.min_confidence,
            "fallbackMode": self.fallback,
        }

    def __repr__(self) -> str:
        return f'calls_on("{self.receiver_type}", "{self.method}")'


def calls_on(
    receiver_type: str,
    method: str,
    min_confidence: float = 0.5,
    fallback: str = "name",
) -> TypeConstrainedCallMatcher:
    """
    Create a matcher for method calls on objects of a specific type.

    Args:
        receiver_type: Type name to match. Supports:
            - Exact: "sqlite3.Cursor"
            - Short name: "Cursor" (matches "sqlite3.Cursor")
            - Wildcard prefix: "*Cursor"
            - Wildcard suffix: "sqlite3.*"
        method: Method name to match (e.g., "execute")
        min_confidence: Minimum type inference confidence (0.0-1.0)
        fallback: Behavior when type info unavailable:
            - "name": match by method name only (default)
            - "none": skip (no match)
            - "warn": match but flag as low confidence

    Returns:
        TypeConstrainedCallMatcher instance

    Examples:
        >>> calls_on("Cursor", "execute")
        calls_on("Cursor", "execute")

        >>> calls_on("sqlite3.Cursor", "execute", min_confidence=0.8)
        calls_on("sqlite3.Cursor", "execute")

        >>> calls_on("Cursor", "execute", fallback="none")
        calls_on("Cursor", "execute")
    """
    return TypeConstrainedCallMatcher(receiver_type, method, min_confidence, fallback)


class ReturnTypeCallMatcher:
    """
    Match functions that return a specific type.

    NOTE: Go-side executor is deferred to a future stack. This matcher
    will serialize valid IR but will not execute on the Go engine yet.
    Use calls_on() for type-aware matching in the current release.

    Examples:
        calls_returning("str")                  # Functions returning str
        calls_returning("List", min_confidence=0.8)  # Higher confidence
    """

    def __init__(self, return_type: str, min_confidence: float = 0.5):
        """
        Args:
            return_type: Expected return type name.
            min_confidence: Minimum confidence threshold (0.0-1.0).

        Raises:
            ValueError: If return_type is empty or min_confidence out of range.
        """
        if not return_type or not isinstance(return_type, str):
            raise ValueError("return_type must be a non-empty string")
        if not isinstance(min_confidence, (int, float)) or not (
            0.0 <= min_confidence <= 1.0
        ):
            raise ValueError(
                f"min_confidence must be 0.0-1.0, got {min_confidence}"
            )

        self.return_type = return_type
        self.min_confidence = float(min_confidence)

    def to_ir(self) -> dict:
        """
        Serialize to JSON IR.

        Returns:
            {
                "type": "return_type_call",
                "returnType": "str",
                "minConfidence": 0.5
            }
        """
        return {
            "type": IRType.RETURN_TYPE_CALL.value,
            "returnType": self.return_type,
            "minConfidence": self.min_confidence,
        }

    def __repr__(self) -> str:
        return f'calls_returning("{self.return_type}")'


def calls_returning(
    return_type: str, min_confidence: float = 0.5
) -> ReturnTypeCallMatcher:
    """
    Match any function call that returns the specified type.

    NOTE: Go-side executor deferred to a future stack. See calls_on()
    for working type-aware matching in the current release.

    Args:
        return_type: Expected return type name.
        min_confidence: Minimum confidence threshold (0.0-1.0).

    Returns:
        ReturnTypeCallMatcher instance

    Examples:
        >>> calls_returning("str")
        calls_returning("str")

        >>> calls_returning("List", min_confidence=0.8)
        calls_returning("List")
    """
    return ReturnTypeCallMatcher(return_type, min_confidence)
