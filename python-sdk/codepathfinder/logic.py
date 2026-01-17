"""Logic operators for combining matchers."""

from typing import Union
from .matchers import CallMatcher, VariableMatcher
from .dataflow import DataflowMatcher
from .ir import IRType

MatcherType = Union[
    CallMatcher,
    VariableMatcher,
    DataflowMatcher,
    "AndOperator",
    "OrOperator",
    "NotOperator",
]


class AndOperator:
    """
    Logical AND - all matchers must match.

    Example:
        And(calls("eval"), variable("user_input"))
        # Matches code that has BOTH eval calls AND user_input variable
    """

    def __init__(self, *matchers: MatcherType):
        if len(matchers) < 2:
            raise ValueError("And() requires at least 2 matchers")
        self.matchers = list(matchers)

    def to_ir(self) -> dict:
        return {
            "type": IRType.LOGIC_AND.value,
            "matchers": [m.to_ir() for m in self.matchers],
        }

    def __repr__(self) -> str:
        return f"And({len(self.matchers)} matchers)"


class OrOperator:
    """
    Logical OR - at least one matcher must match.

    Example:
        Or(calls("eval"), calls("exec"))
        # Matches code with eval OR exec
    """

    def __init__(self, *matchers: MatcherType):
        if len(matchers) < 2:
            raise ValueError("Or() requires at least 2 matchers")
        self.matchers = list(matchers)

    def to_ir(self) -> dict:
        return {
            "type": IRType.LOGIC_OR.value,
            "matchers": [m.to_ir() for m in self.matchers],
        }

    def __repr__(self) -> str:
        return f"Or({len(self.matchers)} matchers)"


class NotOperator:
    """
    Logical NOT - matcher must NOT match.

    Example:
        Not(calls("test_*"))
        # Matches code that does NOT call test_* functions
    """

    def __init__(self, matcher: MatcherType):
        self.matcher = matcher

    def to_ir(self) -> dict:
        return {
            "type": IRType.LOGIC_NOT.value,
            "matcher": self.matcher.to_ir(),
        }

    def __repr__(self) -> str:
        return f"Not({repr(self.matcher)})"


# Public API
def And(*matchers: MatcherType) -> AndOperator:
    """Create AND combinator."""
    return AndOperator(*matchers)


def Or(*matchers: MatcherType) -> OrOperator:
    """Create OR combinator."""
    return OrOperator(*matchers)


def Not(matcher: MatcherType) -> NotOperator:
    """Create NOT combinator."""
    return NotOperator(matcher)
