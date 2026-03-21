"""QueryType: first-class type-aware code queries.

Types are first-class citizens. Define a type once, get precise method matching everywhere.

Example:
    class DBCursor(QueryType):
        fqns = ["sqlite3.Cursor", "psycopg2.extensions.cursor"]
        patterns = ["*.Cursor"]

    DBCursor.method("execute").arg("timeout", missing())
"""

from .qualifiers import Qualifier
from .ir import IRType


class MethodMatcher:
    """Matcher for method calls on typed receivers. Returned by QueryType.method()."""

    def __init__(
        self, receiver_types, receiver_patterns, match_subclasses, method_names
    ):
        self.receiver_types = receiver_types
        self.receiver_patterns = receiver_patterns
        self.match_subclasses = match_subclasses
        self.method_names = method_names
        self.positional_args = {}
        self.keyword_args = {}
        self._tracked_params = []

    def where(self, position_or_name, constraint=None) -> "MethodMatcher":
        """Filter call sites by argument value. Chainable.

        Args:
            position_or_name: int for positional, str for keyword argument
            constraint: expected value, Qualifier (lt, gt, regex, missing), or None
        """
        ac = self._make_constraint(constraint)
        if isinstance(position_or_name, int):
            self.positional_args[str(position_or_name)] = ac
        else:
            self.keyword_args[position_or_name] = ac
        return self

    def arg(self, position_or_name, constraint=None) -> "MethodMatcher":
        """Alias for .where() — backward compatibility."""
        return self.where(position_or_name, constraint)

    def tracks(self, *positions_or_names) -> "MethodMatcher":
        """Specify which parameters are taint-sensitive in dataflow analysis.

        When used on a sink in flows(), only detections where tainted data
        reaches a tracked parameter will be reported.

        When not called, ALL parameters are considered taint-sensitive.

        Args:
            *positions_or_names: int for positional index, str for parameter name,
                                 or "return" for return value tracking.
        """
        for p in positions_or_names:
            if isinstance(p, int):
                self._tracked_params.append({"index": p})
            elif p == "return":
                self._tracked_params.append({"return": True})
            elif isinstance(p, str):
                self._tracked_params.append({"name": p})
            else:
                raise TypeError(
                    f"tracks() accepts int, str, or 'return', got {type(p)}"
                )
        return self

    def _make_constraint(self, value):
        """Convert value or qualifier to ArgumentConstraint dict."""
        if isinstance(value, Qualifier):
            return value.to_constraint()
        has_wildcard = isinstance(value, str) and ("*" in value or "?" in value)
        return {"value": value, "wildcard": has_wildcard}

    def to_ir(self) -> dict:
        ir = {
            "type": IRType.TYPE_CONSTRAINED_CALL.value,
            "receiverTypes": self.receiver_types,
            "receiverPatterns": self.receiver_patterns,
            "matchSubclasses": self.match_subclasses,
            "methodNames": self.method_names,
            "minConfidence": 0.5,
            "fallbackMode": "none",
        }
        if self.positional_args:
            ir["positionalArgs"] = self.positional_args
        if self.keyword_args:
            ir["keywordArgs"] = self.keyword_args
        if self._tracked_params:
            ir["trackedParams"] = self._tracked_params
        return ir

    def __repr__(self) -> str:
        methods = ", ".join(f'"{m}"' for m in self.method_names)
        return f"MethodMatcher({methods})"


class QueryTypeMeta(type):
    """Metaclass for QueryType that provides classmethod-style .method() without instantiation."""

    fqns: list[str]
    patterns: list[str]
    match_subclasses: bool

    def method(cls, *method_names: str) -> MethodMatcher:
        """Select methods to match on this type.

        Args:
            *method_names: One or more method names to match.

        Returns:
            MethodMatcher configured with this type's FQNs and patterns.
        """
        if not method_names:
            raise ValueError("method() requires at least one method name")
        return MethodMatcher(
            receiver_types=cls.fqns,
            receiver_patterns=cls.patterns,
            match_subclasses=cls.match_subclasses,
            method_names=list(method_names),
        )


class QueryType(metaclass=QueryTypeMeta):
    """Base class for type-aware code queries. Types are first-class.

    Example:
        class DBCursor(QueryType):
            fqns = ["sqlite3.Cursor", "psycopg2.extensions.cursor"]
            patterns = ["*.Cursor"]
            match_subclasses = True

        DBCursor.method("execute", "executemany")
    """

    fqns: list = []
    patterns: list = []
    match_subclasses: bool = True
