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

from .matchers import calls, variable, attribute
from .decorators import rule
from .dataflow import flows
from .propagation import propagates
from .presets import PropagationPresets
from .config import set_default_propagation, set_default_scope
from .logic import And, Or, Not
from .query_type import QueryType
from .qualifiers import lt, gt, lte, gte, regex, missing

__all__ = [
    "attribute",
    "calls",
    "variable",
    "rule",
    "flows",
    "propagates",
    "PropagationPresets",
    "set_default_propagation",
    "set_default_scope",
    "And",
    "Or",
    "Not",
    "QueryType",
    "lt",
    "gt",
    "lte",
    "gte",
    "regex",
    "missing",
    "__version__",
]
