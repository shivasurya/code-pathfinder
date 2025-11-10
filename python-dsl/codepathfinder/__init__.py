"""
codepathfinder - Python DSL for static analysis security patterns

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

__version__ = "1.0.0"

from .matchers import calls, variable
from .decorators import rule
from .dataflow import flows
from .propagation import propagates

__all__ = ["calls", "variable", "rule", "flows", "propagates", "__version__"]
