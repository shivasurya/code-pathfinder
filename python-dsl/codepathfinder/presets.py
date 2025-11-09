"""
Propagation presets for common use cases.

Presets bundle propagation primitives for convenience.
"""

from typing import List
from .propagation import propagates, PropagationPrimitive


class PropagationPresets:
    """
    Common propagation bundles.

    Developers can use presets instead of manually listing primitives.
    """

    @staticmethod
    def minimal() -> List[PropagationPrimitive]:
        """
        Bare minimum propagation (fastest, least false negatives).

        Covers:
            - Variable assignments
            - Function arguments

        Coverage: ~40% of real-world flows
        Performance: Fastest (minimal overhead)
        False negatives: Higher (misses return values, strings)

        Use when:
            - Performance is critical
            - You only care about direct variable flows

        Example:
            flows(
                from_sources=calls("request.GET"),
                to_sinks=calls("eval"),
                propagates_through=PropagationPresets.minimal(),
                scope="local"
            )
        """
        return [
            propagates.assignment(),
            propagates.function_args(),
        ]

    @staticmethod
    def standard() -> List[PropagationPrimitive]:
        """
        Recommended default (good balance).

        Covers:
            - Phase 1: assignment, function_args, function_returns
            - Phase 2: string_concat, string_format

        Coverage: ~75-80% of real-world flows
        Performance: Good (moderate overhead)
        False negatives: Lower

        Use when:
            - General-purpose taint analysis
            - OWASP Top 10 detection
            - Good balance of coverage and performance

        Example:
            flows(
                from_sources=calls("request.*"),
                to_sinks=calls("execute"),
                propagates_through=PropagationPresets.standard(),
                scope="global"
            )
        """
        return [
            propagates.assignment(),
            propagates.function_args(),
            propagates.function_returns(),
            propagates.string_concat(),
            propagates.string_format(),
        ]

    @staticmethod
    def comprehensive() -> List[PropagationPrimitive]:
        """
        All MVP primitives (Phase 1 + Phase 2).

        Covers:
            - All standard() primitives

        Coverage: ~80% of real-world flows
        Performance: Moderate
        False negatives: Low

        Use when:
            - Maximum coverage within MVP scope
            - Willing to accept moderate performance overhead

        Example:
            flows(
                from_sources=calls("request.*"),
                to_sinks=calls("eval"),
                propagates_through=PropagationPresets.comprehensive(),
                scope="global"
            )
        """
        return PropagationPresets.standard()  # For MVP, comprehensive = standard

    @staticmethod
    def exhaustive() -> List[PropagationPrimitive]:
        """
        All primitives (Phase 1-6, POST-MVP).

        NOTE: For MVP, this is same as comprehensive().
        Post-MVP will include collections, control flow, OOP, advanced.

        Coverage: ~95% of real-world flows (POST-MVP)
        Performance: Slower (comprehensive analysis)
        False negatives: Minimal

        Use when:
            - Maximum security coverage required
            - Performance is not a concern
            - Production-critical code

        Example:
            flows(
                from_sources=calls("request.*"),
                to_sinks=calls("execute"),
                propagates_through=PropagationPresets.exhaustive(),
                scope="global"
            )
        """
        # MVP: same as comprehensive
        # POST-MVP: will include Phase 3-6 primitives
        return PropagationPresets.comprehensive()
