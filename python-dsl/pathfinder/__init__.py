"""
pathfinder - Python DSL for static analysis security patterns

Examples:
    Basic matchers:
        >>> from pathfinder import calls, variable
        >>> calls("eval")
        >>> variable("user_input")

    Rule definition:
        >>> from pathfinder import rule, calls
        >>> @rule(id="test", severity="high")
        >>> def detect_eval():
        >>>     return calls("eval")
"""

__version__ = "1.0.0"

from .matchers import calls, variable
from .decorators import rule

__all__ = ["calls", "variable", "rule", "__version__"]
