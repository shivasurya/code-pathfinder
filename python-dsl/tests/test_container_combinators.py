"""Tests for logic combinators."""

from rules.container_matchers import instruction, missing
from rules.container_combinators import (
    all_of,
    any_of,
    none_of,
    instruction_after,
    instruction_before,
    stage,
    final_stage_has,
)


class TestAllOf:
    def test_basic(self):
        m = all_of(
            instruction(type="FROM", image_tag="latest"), missing(instruction="USER")
        )
        d = m.to_dict()
        assert d["type"] == "all_of"
        assert len(d["conditions"]) == 2

    def test_nested(self):
        m = all_of(
            any_of(
                instruction(type="USER", user_name="root"), missing(instruction="USER")
            ),
            instruction(type="FROM"),
        )
        d = m.to_dict()
        assert d["conditions"][0]["type"] == "any_of"

    def test_multiple_conditions(self):
        m = all_of(
            instruction(type="FROM", image_tag="latest"),
            missing(instruction="USER"),
            instruction(type="RUN", contains="sudo"),
        )
        d = m.to_dict()
        assert len(d["conditions"]) == 3

    def test_with_dict(self):
        m = all_of({"type": "custom", "value": "test"}, instruction(type="FROM"))
        d = m.to_dict()
        assert d["conditions"][0]["type"] == "custom"

    def test_with_callable(self):
        def custom_func():
            return True

        m = all_of(custom_func, instruction(type="FROM"))
        d = m.to_dict()
        assert d["conditions"][0]["type"] == "custom_function"
        assert d["conditions"][0]["has_callable"] is True


class TestAnyOf:
    def test_basic(self):
        m = any_of(
            instruction(type="FROM", image_tag="latest"),
            instruction(type="FROM", base_image="scratch"),
        )
        d = m.to_dict()
        assert d["type"] == "any_of"
        assert len(d["conditions"]) == 2

    def test_single_condition(self):
        m = any_of(instruction(type="USER", user_name="root"))
        d = m.to_dict()
        assert len(d["conditions"]) == 1

    def test_many_conditions(self):
        m = any_of(
            instruction(type="USER", user_name="root"),
            missing(instruction="USER"),
            instruction(type="FROM", base_image="scratch"),
            instruction(type="RUN", contains="sudo"),
        )
        d = m.to_dict()
        assert len(d["conditions"]) == 4


class TestNoneOf:
    def test_basic(self):
        m = none_of(instruction(type="HEALTHCHECK"))
        d = m.to_dict()
        assert d["type"] == "none_of"

    def test_multiple_conditions(self):
        m = none_of(instruction(type="HEALTHCHECK"), instruction(type="USER"))
        d = m.to_dict()
        assert len(d["conditions"]) == 2


class TestInstructionSequence:
    def test_after_string(self):
        m = instruction_after(instruction="CMD", after="USER")
        d = m.to_dict()
        assert d["type"] == "instruction_after"
        assert d["instruction"]["instruction"] == "CMD"
        assert d["reference"]["instruction"] == "USER"

    def test_before_string(self):
        m = instruction_before(instruction="USER", before="CMD")
        d = m.to_dict()
        assert d["type"] == "instruction_before"

    def test_with_matcher(self):
        m = instruction_after(
            instruction=instruction(type="RUN", contains="apt-get install"),
            after=instruction(type="RUN", contains="apt-get update"),
        )
        d = m.to_dict()
        assert "contains" in d["instruction"]
        assert "contains" in d["reference"]

    def test_not_followed_by(self):
        m = instruction_after(instruction="RUN", after="FROM", not_followed_by=True)
        d = m.to_dict()
        assert d["not_followed_by"] is True

    def test_before_with_matcher(self):
        m = instruction_before(
            instruction=instruction(type="USER", user_name="root"), before="CMD"
        )
        d = m.to_dict()
        assert d["instruction"]["user_name"] == "root"

    def test_with_dict(self):
        m = instruction_after(instruction={"type": "custom"}, after={"type": "other"})
        d = m.to_dict()
        assert d["instruction"]["type"] == "custom"
        assert d["reference"]["type"] == "other"


class TestStageMatcher:
    def test_stage_alias(self):
        m = stage(alias="builder")
        d = m.to_dict()
        assert d["type"] == "stage_query"
        assert d["alias"] == "builder"

    def test_final_stage(self):
        m = stage(is_final=True)
        d = m.to_dict()
        assert d["is_final"] is True

    def test_stage_base_image(self):
        m = stage(base_image="alpine")
        d = m.to_dict()
        assert d["base_image"] == "alpine"

    def test_all_params(self):
        m = stage(alias="builder", base_image="alpine", is_final=False)
        d = m.to_dict()
        assert d["alias"] == "builder"
        assert d["base_image"] == "alpine"
        assert d["is_final"] is False

    def test_final_stage_has_missing(self):
        m = final_stage_has(missing_instruction="USER")
        d = m.to_dict()
        assert d["type"] == "stage_final_has"
        assert d["missing_instruction"] == "USER"

    def test_final_stage_has_instruction_string(self):
        m = final_stage_has(instruction="USER")
        d = m.to_dict()
        assert d["instruction"] == "USER"

    def test_final_stage_has_instruction_matcher(self):
        m = final_stage_has(instruction=instruction(type="USER", user_name="root"))
        d = m.to_dict()
        assert d["instruction"]["user_name"] == "root"

    def test_final_stage_has_both_params(self):
        m = final_stage_has(
            instruction=instruction(type="RUN"), missing_instruction="HEALTHCHECK"
        )
        d = m.to_dict()
        assert "instruction" in d
        assert d["missing_instruction"] == "HEALTHCHECK"
