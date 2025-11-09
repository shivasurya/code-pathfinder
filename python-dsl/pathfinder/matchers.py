"""
Core matchers for the pathfinder Python DSL.

These matchers generate JSON IR for the Go executor.
"""

from .ir import IRType


class CallMatcher:
    """
    Matches function/method calls in the callgraph.

    Examples:
        calls("eval")                    # Exact match
        calls("eval", "exec")            # Multiple patterns
        calls("request.*")               # Wildcard (any request.* call)
        calls("*.json")                  # Wildcard (any *.json call)
    """

    def __init__(self, *patterns: str):
        """
        Args:
            *patterns: Function names to match. Supports wildcards (*).

        Raises:
            ValueError: If no patterns provided or pattern is empty
        """
        if not patterns:
            raise ValueError("calls() requires at least one pattern")

        if any(not p or not isinstance(p, str) for p in patterns):
            raise ValueError("All patterns must be non-empty strings")

        self.patterns = list(patterns)
        self.wildcard = any("*" in p for p in patterns)

    def to_ir(self) -> dict:
        """
        Serialize to JSON IR for Go executor.

        Returns:
            {
                "type": "call_matcher",
                "patterns": ["eval", "exec"],
                "wildcard": false,
                "match_mode": "any"  # matches if ANY pattern matches
            }
        """
        return {
            "type": IRType.CALL_MATCHER.value,
            "patterns": self.patterns,
            "wildcard": self.wildcard,
            "match_mode": "any",
        }

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
def calls(*patterns: str) -> CallMatcher:
    """
    Create a matcher for function/method calls.

    Args:
        *patterns: Function names to match (supports wildcards)

    Returns:
        CallMatcher instance

    Examples:
        >>> calls("eval")
        calls("eval")

        >>> calls("request.GET", "request.POST")
        calls("request.GET", "request.POST")

        >>> calls("urllib.*")
        calls("urllib.*")
    """
    return CallMatcher(*patterns)


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
