"""
codepathfinder - Python SDK for static analysis security patterns

Examples:
    Basic matchers:
        >>> from codepathfinder import calls, variable
        >>> calls("eval")
        >>> variable("user_input")

    Rule definition:
        >>> from codepathfinder import rule, calls
        >>> @rule(id="test", severity="high")
        >>> def detect_eval():
        >>>     return calls("eval")

    Dataflow analysis:
        >>> from codepathfinder import flows, calls, propagates
        >>> flows(
        ...     from_sources=calls("request.GET"),
        ...     to_sinks=calls("execute"),
        ...     propagates_through=[propagates.assignment()]
        ... )
"""

__version__ = "2.1.1"

from .config import set_default_propagation, set_default_scope
from .dataflow import flows
from .decorators import rule
from .logic import And, Not, Or
from .matchers import attribute, calls, variable
from .presets import PropagationPresets
from .propagation import propagates
from .qualifiers import gt, gte, lt, lte, missing, regex
from .query_type import QueryType

__all__ = [
    "And",
    "Not",
    "Or",
    "PropagationPresets",
    "QueryType",
    "__version__",
    "attribute",
    "calls",
    "flows",
    "gt",
    "gte",
    "lt",
    "lte",
    "missing",
    "propagates",
    "regex",
    "rule",
    "set_default_propagation",
    "set_default_scope",
    "variable",
]
