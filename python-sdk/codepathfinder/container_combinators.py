"""
Logic combinators for container rules.
"""

from typing import List, Dict, Any, Union, Callable
from dataclasses import dataclass, field
from .container_matchers import Matcher


@dataclass
class CombinatorMatcher:
    """Represents a logic combinator (AND, OR, NOT)."""

    combinator_type: str  # "all_of", "any_of", "none_of"
    conditions: List[Union[Matcher, "CombinatorMatcher", Dict, Callable]]

    def to_dict(self) -> Dict[str, Any]:
        """Convert to JSON IR."""
        serialized_conditions = []
        for cond in self.conditions:
            if hasattr(cond, "to_dict"):
                serialized_conditions.append(cond.to_dict())
            elif isinstance(cond, dict):
                serialized_conditions.append(cond)
            elif callable(cond):
                serialized_conditions.append(
                    {"type": "custom_function", "has_callable": True}
                )
            else:
                serialized_conditions.append(cond)

        return {"type": self.combinator_type, "conditions": serialized_conditions}


def all_of(*conditions: Union[Matcher, Dict, Callable]) -> CombinatorMatcher:
    """
    Combine matchers with AND logic.
    All conditions must match for the rule to trigger.

    Example:
        all_of(
            instruction(type="FROM", image_tag="latest"),
            missing(instruction="USER"),
            instruction(type="RUN", contains="sudo")
        )
    """
    return CombinatorMatcher(combinator_type="all_of", conditions=list(conditions))


def any_of(*conditions: Union[Matcher, Dict, Callable]) -> CombinatorMatcher:
    """
    Combine matchers with OR logic.
    Any condition can match for the rule to trigger.

    Example:
        any_of(
            instruction(type="USER", user_name="root"),
            missing(instruction="USER"),
            instruction(type="FROM", base_image="scratch")
        )
    """
    return CombinatorMatcher(combinator_type="any_of", conditions=list(conditions))


def none_of(*conditions: Union[Matcher, Dict, Callable]) -> CombinatorMatcher:
    """
    Combine matchers with NOT logic.
    None of the conditions should match for the rule to pass.
    (Inverse: if any matches, rule triggers as violation)

    Example:
        none_of(
            instruction(type="HEALTHCHECK"),
            instruction(type="USER", user_name_not="root")
        )
    """
    return CombinatorMatcher(combinator_type="none_of", conditions=list(conditions))


@dataclass
class SequenceMatcher:
    """Represents instruction sequence validation."""

    sequence_type: str  # "after" or "before"
    instruction: Union[str, Matcher, Dict]
    reference: Union[str, Matcher, Dict]
    not_followed_by: bool = False

    def to_dict(self) -> Dict[str, Any]:
        """Convert to JSON IR."""

        def serialize_ref(ref):
            if isinstance(ref, str):
                return {"instruction": ref}
            elif hasattr(ref, "to_dict"):
                return ref.to_dict()
            elif isinstance(ref, dict):
                return ref
            return ref

        return {
            "type": f"instruction_{self.sequence_type}",
            "instruction": serialize_ref(self.instruction),
            "reference": serialize_ref(self.reference),
            "not_followed_by": self.not_followed_by,
        }


def instruction_after(
    instruction: Union[str, Matcher],
    after: Union[str, Matcher],
    not_followed_by: bool = False,
) -> SequenceMatcher:
    """
    Check that an instruction appears after another.

    Example:
        # Ensure CMD comes after USER
        instruction_after(instruction="CMD", after="USER")

        # Ensure apt-get install follows apt-get update
        instruction_after(
            instruction=instruction(type="RUN", contains="apt-get install"),
            after=instruction(type="RUN", contains="apt-get update")
        )
    """
    return SequenceMatcher(
        sequence_type="after",
        instruction=instruction,
        reference=after,
        not_followed_by=not_followed_by,
    )


def instruction_before(
    instruction: Union[str, Matcher],
    before: Union[str, Matcher],
    not_followed_by: bool = False,
) -> SequenceMatcher:
    """
    Check that an instruction appears before another.

    Example:
        instruction_before(instruction="USER", before="CMD")
    """
    return SequenceMatcher(
        sequence_type="before",
        instruction=instruction,
        reference=before,
        not_followed_by=not_followed_by,
    )


@dataclass
class StageMatcher:
    """Matcher for multi-stage build stage queries."""

    stage_type: str
    params: Dict[str, Any] = field(default_factory=dict)

    def to_dict(self) -> Dict[str, Any]:
        return {"type": f"stage_{self.stage_type}", **self.params}


def stage(
    alias: str = None,
    base_image: str = None,
    is_final: bool = None,
) -> StageMatcher:
    """
    Query a specific build stage.

    Example:
        stage(alias="builder")
        stage(is_final=True)
        stage(base_image="alpine")
    """
    params = {}
    if alias is not None:
        params["alias"] = alias
    if base_image is not None:
        params["base_image"] = base_image
    if is_final is not None:
        params["is_final"] = is_final

    return StageMatcher(stage_type="query", params=params)


def final_stage_has(
    instruction: Union[str, Matcher] = None,
    missing_instruction: str = None,
) -> StageMatcher:
    """
    Check properties of the final build stage.

    Example:
        final_stage_has(missing_instruction="USER")
        final_stage_has(instruction=instruction(type="USER", user_name="root"))
    """
    params = {}
    if instruction is not None:
        if isinstance(instruction, str):
            params["instruction"] = instruction
        elif hasattr(instruction, "to_dict"):
            params["instruction"] = instruction.to_dict()
    if missing_instruction is not None:
        params["missing_instruction"] = missing_instruction

    return StageMatcher(stage_type="final_has", params=params)
