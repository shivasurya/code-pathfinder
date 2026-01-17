"""Tests for container rule decorators."""

import pytest
from rules.container_decorators import (
    dockerfile_rule,
    compose_rule,
    get_dockerfile_rules,
    get_compose_rules,
    clear_rules,
)
from rules.container_matchers import (
    instruction,
    missing,
    service_has,
    service_missing,
)


class TestDockerfileRule:
    def setup_method(self):
        clear_rules()

    def test_basic_rule(self):
        @dockerfile_rule(id="TEST-001", severity="HIGH")
        def test_rule():
            return missing(instruction="USER")

        rules = get_dockerfile_rules()
        assert len(rules) == 1
        assert rules[0].metadata.id == "TEST-001"
        assert rules[0].metadata.severity == "HIGH"

    def test_rule_with_all_metadata(self):
        @dockerfile_rule(
            id="TEST-002",
            name="Test Rule Name",
            severity="CRITICAL",
            category="best-practice",
            cwe="CWE-250",
            message="Custom message",
        )
        def custom_rule():
            return instruction(type="FROM", image_tag="latest")

        rules = get_dockerfile_rules()
        assert rules[0].metadata.name == "Test Rule Name"
        assert rules[0].metadata.category == "best-practice"
        assert rules[0].metadata.cwe == "CWE-250"
        assert rules[0].metadata.message == "Custom message"

    def test_matcher_conversion(self):
        @dockerfile_rule(id="TEST-003")
        def matcher_test():
            return instruction(type="USER", user_name="root")

        rules = get_dockerfile_rules()
        matcher = rules[0].matcher
        assert matcher["type"] == "instruction"
        assert matcher["instruction"] == "USER"
        assert matcher["user_name"] == "root"

    def test_invalid_matcher_type(self):
        with pytest.raises(ValueError) as excinfo:

            @dockerfile_rule(id="TEST-004")
            def bad_rule():
                return "not a matcher"

        assert "must return a matcher or dict" in str(excinfo.value)

    def test_dict_matcher(self):
        @dockerfile_rule(id="TEST-005")
        def dict_rule():
            return {"type": "custom", "param": "value"}

        rules = get_dockerfile_rules()
        assert rules[0].matcher["type"] == "custom"
        assert rules[0].matcher["param"] == "value"


class TestComposeRule:
    def setup_method(self):
        clear_rules()

    def test_basic_rule(self):
        @compose_rule(id="COMPOSE-001", severity="HIGH")
        def privileged_service():
            return service_has(key="privileged", equals=True)

        rules = get_compose_rules()
        assert len(rules) == 1
        assert rules[0].metadata.id == "COMPOSE-001"
        assert rules[0].metadata.file_pattern == "**/docker-compose*.yml"

    def test_service_missing(self):
        @compose_rule(id="COMPOSE-002")
        def no_read_only():
            return service_missing(key="read_only")

        rules = get_compose_rules()
        matcher = rules[0].matcher
        assert matcher["type"] == "service_missing"
        assert matcher["key"] == "read_only"

    def test_invalid_matcher_type(self):
        with pytest.raises(ValueError) as excinfo:

            @compose_rule(id="COMPOSE-003")
            def bad_rule():
                return "not a matcher"

        assert "must return a matcher or dict" in str(excinfo.value)

    def test_dict_matcher(self):
        @compose_rule(id="COMPOSE-004")
        def dict_rule():
            return {"type": "custom", "key": "value"}

        rules = get_compose_rules()
        assert rules[0].matcher["type"] == "custom"
        assert rules[0].matcher["key"] == "value"
