"""
Dataflow matcher for taint analysis.

The flows() function is the core of OWASP Top 10 pattern detection.
It describes how tainted data flows from sources to sinks.
"""

from typing import List, Optional, Union
from .matchers import CallMatcher
from .propagation import PropagationPrimitive, create_propagation_list
from .ir import IRType
from .config import get_default_propagation, get_default_scope


class DataflowMatcher:
    """
    Matches tainted data flows from sources to sinks.

    This is the primary matcher for security vulnerabilities like:
    - SQL Injection (A03:2021)
    - Command Injection (A03:2021)
    - SSRF (A10:2021)
    - Path Traversal (A01:2021)
    - Insecure Deserialization (A08:2021)

    Attributes:
        sources: Matchers for taint sources (e.g., user input)
        sinks: Matchers for dangerous sinks (e.g., eval, execute)
        sanitizers: Optional matchers for sanitizer functions
        propagates_through: List of propagation primitives (EXPLICIT!)
        scope: "local" (same function) or "global" (cross-function)
    """

    def __init__(
        self,
        from_sources: Union[CallMatcher, List[CallMatcher]],
        to_sinks: Union[CallMatcher, List[CallMatcher]],
        sanitized_by: Optional[Union[CallMatcher, List[CallMatcher]]] = None,
        propagates_through: Optional[List[PropagationPrimitive]] = None,
        scope: Optional[str] = None,
    ):
        """
        Args:
            from_sources: Source matcher(s) - where taint originates
            to_sinks: Sink matcher(s) - dangerous functions
            sanitized_by: Optional sanitizer matcher(s)
            propagates_through: EXPLICIT list of propagation primitives
                                (default: None = no propagation!)
            scope: "local" (intra-procedural) or "global" (inter-procedural)

        Raises:
            ValueError: If sources/sinks are empty, scope invalid, etc.

        Examples:
            # SQL Injection
            flows(
                from_sources=calls("request.GET", "request.POST"),
                to_sinks=calls("execute", "executemany"),
                sanitized_by=calls("quote_sql"),
                propagates_through=[
                    propagates.assignment(),
                    propagates.function_args(),
                ],
                scope="global"
            )
        """
        # Validate sources
        if isinstance(from_sources, CallMatcher):
            from_sources = [from_sources]
        if not from_sources:
            raise ValueError("flows() requires at least one source")
        self.sources = from_sources

        # Validate sinks
        if isinstance(to_sinks, CallMatcher):
            to_sinks = [to_sinks]
        if not to_sinks:
            raise ValueError("flows() requires at least one sink")
        self.sinks = to_sinks

        # Validate sanitizers
        if sanitized_by is None:
            sanitized_by = []
        elif isinstance(sanitized_by, CallMatcher):
            sanitized_by = [sanitized_by]
        self.sanitizers = sanitized_by

        # Validate propagation (use global default if not specified)
        if propagates_through is None:
            propagates_through = get_default_propagation()
        self.propagates_through = propagates_through

        # Validate scope (use global default if not specified)
        if scope is None:
            scope = get_default_scope()
        if scope not in ["local", "global"]:
            raise ValueError(f"scope must be 'local' or 'global', got '{scope}'")
        self.scope = scope

    def to_ir(self) -> dict:
        """
        Serialize to JSON IR for Go executor.

        Returns:
            {
                "type": "dataflow",
                "sources": [
                    {"type": "call_matcher", "patterns": ["request.GET"], ...}
                ],
                "sinks": [
                    {"type": "call_matcher", "patterns": ["execute"], ...}
                ],
                "sanitizers": [
                    {"type": "call_matcher", "patterns": ["quote_sql"], ...}
                ],
                "propagation": [
                    {"type": "assignment", "metadata": {}},
                    {"type": "function_args", "metadata": {}}
                ],
                "scope": "global"
            }
        """
        return {
            "type": IRType.DATAFLOW.value,
            "sources": [src.to_ir() for src in self.sources],
            "sinks": [sink.to_ir() for sink in self.sinks],
            "sanitizers": [san.to_ir() for san in self.sanitizers],
            "propagation": create_propagation_list(self.propagates_through),
            "scope": self.scope,
        }

    def __repr__(self) -> str:
        src_count = len(self.sources)
        sink_count = len(self.sinks)
        prop_count = len(self.propagates_through)
        return (
            f"flows(sources={src_count}, sinks={sink_count}, "
            f"propagation={prop_count}, scope='{self.scope}')"
        )


# Public API
def flows(
    from_sources: Union[CallMatcher, List[CallMatcher]],
    to_sinks: Union[CallMatcher, List[CallMatcher]],
    sanitized_by: Optional[Union[CallMatcher, List[CallMatcher]]] = None,
    propagates_through: Optional[List[PropagationPrimitive]] = None,
    scope: Optional[str] = None,
) -> DataflowMatcher:
    """
    Create a dataflow matcher for taint analysis.

    This is the PRIMARY matcher for OWASP Top 10 vulnerabilities.

    Args:
        from_sources: Where taint originates (e.g., user input)
        to_sinks: Dangerous functions that consume tainted data
        sanitized_by: Optional functions that neutralize taint
        propagates_through: HOW taint flows (MUST be explicit!)
        scope: "local" or "global" analysis

    Returns:
        DataflowMatcher instance

    Examples:
        >>> from codepathfinder import flows, calls, propagates
        >>>
        >>> # SQL Injection
        >>> flows(
        ...     from_sources=calls("request.GET"),
        ...     to_sinks=calls("execute"),
        ...     propagates_through=[propagates.assignment()]
        ... )
        >>>
        >>> # Command Injection with sanitization
        >>> flows(
        ...     from_sources=calls("request.POST"),
        ...     to_sinks=calls("os.system", "subprocess.call"),
        ...     sanitized_by=calls("shlex.quote"),
        ...     propagates_through=[
        ...         propagates.assignment(),
        ...         propagates.function_args()
        ...     ],
        ...     scope="global"
        ... )
    """
    return DataflowMatcher(
        from_sources=from_sources,
        to_sinks=to_sinks,
        sanitized_by=sanitized_by,
        propagates_through=propagates_through,
        scope=scope,
    )
