"""Tests for pathfinder.ir module."""

import pytest
from codepathfinder.ir import IRType, serialize_ir, validate_ir
from codepathfinder import calls, variable, flows, propagates


class TestIRType:
    """Test IRType enum."""

    def test_enum_values(self):
        """Test IRType enum has correct values."""
        assert IRType.CALL_MATCHER.value == "call_matcher"
        assert IRType.VARIABLE_MATCHER.value == "variable_matcher"


class TestSerializeIR:
    """Test serialize_ir() function."""

    def test_serialize_call_matcher(self):
        """Test serializing CallMatcher."""
        matcher = calls("eval")
        ir = serialize_ir(matcher)

        assert ir["type"] == "call_matcher"
        assert ir["patterns"] == ["eval"]

    def test_serialize_variable_matcher(self):
        """Test serializing VariableMatcher."""
        matcher = variable("user_input")
        ir = serialize_ir(matcher)

        assert ir["type"] == "variable_matcher"
        assert ir["pattern"] == "user_input"

    def test_serialize_non_matcher_raises(self):
        """Test serializing object without to_ir() raises AttributeError."""
        with pytest.raises(AttributeError, match="must implement to_ir"):
            serialize_ir("not a matcher")  # type: ignore


class TestValidateIR:
    """Test validate_ir() function."""

    def test_valid_call_matcher_ir(self):
        """Test validating valid call_matcher IR."""
        ir = {
            "type": "call_matcher",
            "patterns": ["eval"],
            "wildcard": False,
            "match_mode": "any",
        }
        assert validate_ir(ir) is True

    def test_valid_variable_matcher_ir(self):
        """Test validating valid variable_matcher IR."""
        ir = {
            "type": "variable_matcher",
            "pattern": "user_input",
            "wildcard": False,
        }
        assert validate_ir(ir) is True

    def test_missing_type_field(self):
        """Test IR without 'type' field is invalid."""
        ir = {"patterns": ["eval"]}
        assert validate_ir(ir) is False

    def test_invalid_type_value(self):
        """Test IR with invalid 'type' value is invalid."""
        ir = {"type": "invalid_type", "patterns": ["eval"]}
        assert validate_ir(ir) is False

    def test_call_matcher_missing_patterns(self):
        """Test call_matcher IR without 'patterns' is invalid."""
        ir = {"type": "call_matcher", "wildcard": False}
        assert validate_ir(ir) is False

    def test_call_matcher_empty_patterns(self):
        """Test call_matcher IR with empty patterns list is invalid."""
        ir = {"type": "call_matcher", "patterns": [], "wildcard": False}
        assert validate_ir(ir) is False

    def test_variable_matcher_missing_pattern(self):
        """Test variable_matcher IR without 'pattern' is invalid."""
        ir = {"type": "variable_matcher", "wildcard": False}
        assert validate_ir(ir) is False

    def test_variable_matcher_empty_pattern(self):
        """Test variable_matcher IR with empty pattern is invalid."""
        ir = {"type": "variable_matcher", "pattern": "", "wildcard": False}
        assert validate_ir(ir) is False

    def test_valid_dataflow_ir(self):
        """Test validating valid dataflow IR."""
        ir = {
            "type": "dataflow",
            "sources": [
                {"type": "call_matcher", "patterns": ["source"], "wildcard": False}
            ],
            "sinks": [
                {"type": "call_matcher", "patterns": ["sink"], "wildcard": False}
            ],
            "sanitizers": [],
            "propagation": [],
            "scope": "global",
        }
        assert validate_ir(ir) is True

    def test_dataflow_missing_sources(self):
        """Test dataflow IR without 'sources' is invalid."""
        ir = {
            "type": "dataflow",
            "sinks": [{"type": "call_matcher"}],
            "sanitizers": [],
            "propagation": [],
            "scope": "global",
        }
        assert validate_ir(ir) is False

    def test_dataflow_empty_sources(self):
        """Test dataflow IR with empty sources list is invalid."""
        ir = {
            "type": "dataflow",
            "sources": [],
            "sinks": [{"type": "call_matcher"}],
            "sanitizers": [],
            "propagation": [],
            "scope": "global",
        }
        assert validate_ir(ir) is False

    def test_dataflow_missing_sinks(self):
        """Test dataflow IR without 'sinks' is invalid."""
        ir = {
            "type": "dataflow",
            "sources": [{"type": "call_matcher"}],
            "sanitizers": [],
            "propagation": [],
            "scope": "global",
        }
        assert validate_ir(ir) is False

    def test_dataflow_empty_sinks(self):
        """Test dataflow IR with empty sinks list is invalid."""
        ir = {
            "type": "dataflow",
            "sources": [{"type": "call_matcher"}],
            "sinks": [],
            "sanitizers": [],
            "propagation": [],
            "scope": "global",
        }
        assert validate_ir(ir) is False

    def test_dataflow_invalid_scope(self):
        """Test dataflow IR with invalid scope is invalid."""
        ir = {
            "type": "dataflow",
            "sources": [{"type": "call_matcher"}],
            "sinks": [{"type": "call_matcher"}],
            "sanitizers": [],
            "propagation": [],
            "scope": "invalid",
        }
        assert validate_ir(ir) is False

    def test_dataflow_local_scope_valid(self):
        """Test dataflow IR with local scope is valid."""
        ir = {
            "type": "dataflow",
            "sources": [
                {"type": "call_matcher", "patterns": ["source"], "wildcard": False}
            ],
            "sinks": [
                {"type": "call_matcher", "patterns": ["sink"], "wildcard": False}
            ],
            "sanitizers": [],
            "propagation": [],
            "scope": "local",
        }
        assert validate_ir(ir) is True

    def test_serialize_dataflow_matcher(self):
        """Test serializing DataflowMatcher."""
        matcher = flows(
            from_sources=calls("request.GET"),
            to_sinks=calls("execute"),
            propagates_through=[propagates.assignment()],
        )
        ir = serialize_ir(matcher)

        assert ir["type"] == "dataflow"
        assert len(ir["sources"]) == 1
        assert len(ir["sinks"]) == 1
        assert len(ir["propagation"]) == 1
