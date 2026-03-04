"""Tests for type-aware matchers: calls_on() and calls_returning()."""

import json

import pytest
from codepathfinder import calls_on, calls_returning
from codepathfinder.matchers import TypeConstrainedCallMatcher, ReturnTypeCallMatcher
from codepathfinder.ir import IRType, validate_ir


class TestCallsOn:
    """Test suite for calls_on() matcher."""

    def test_basic_ir(self):
        """Test calls_on() produces correct IR structure."""
        matcher = calls_on("Cursor", "execute")
        ir = matcher.to_ir()

        assert ir["type"] == "type_constrained_call"
        assert ir["receiverType"] == "Cursor"
        assert ir["methodName"] == "execute"
        assert ir["minConfidence"] == 0.5
        assert ir["fallbackMode"] == "name"

    def test_custom_params(self):
        """Test calls_on() with non-default confidence and fallback."""
        matcher = calls_on("Cursor", "execute", min_confidence=0.8, fallback="none")
        ir = matcher.to_ir()

        assert ir["minConfidence"] == 0.8
        assert ir["fallbackMode"] == "none"

    def test_validation_empty_type(self):
        """Test ValueError for empty receiver_type."""
        with pytest.raises(ValueError, match="receiver_type must be a non-empty string"):
            calls_on("", "execute")

    def test_validation_none_type(self):
        """Test ValueError for None receiver_type."""
        with pytest.raises(ValueError, match="receiver_type must be a non-empty string"):
            calls_on(None, "execute")

    def test_validation_empty_method(self):
        """Test ValueError for empty method."""
        with pytest.raises(ValueError, match="method must be a non-empty string"):
            calls_on("Cursor", "")

    def test_validation_none_method(self):
        """Test ValueError for None method."""
        with pytest.raises(ValueError, match="method must be a non-empty string"):
            calls_on("Cursor", None)

    def test_validation_bad_fallback(self):
        """Test ValueError for invalid fallback value."""
        with pytest.raises(ValueError, match="fallback must be"):
            calls_on("Cursor", "execute", fallback="invalid")

    def test_validation_bad_confidence_high(self):
        """Test ValueError for confidence > 1.0."""
        with pytest.raises(ValueError, match="min_confidence must be"):
            calls_on("Cursor", "execute", min_confidence=1.5)

    def test_validation_bad_confidence_negative(self):
        """Test ValueError for negative confidence."""
        with pytest.raises(ValueError, match="min_confidence must be"):
            calls_on("Cursor", "execute", min_confidence=-0.1)

    def test_repr(self):
        """Test string representation."""
        matcher = calls_on("Cursor", "execute")
        assert repr(matcher) == 'calls_on("Cursor", "execute")'

    def test_wildcard_prefix(self):
        """Test wildcard prefix pattern in receiver_type."""
        matcher = calls_on("*Cursor", "execute")
        ir = matcher.to_ir()
        assert ir["receiverType"] == "*Cursor"

    def test_wildcard_suffix(self):
        """Test wildcard suffix pattern in receiver_type."""
        matcher = calls_on("sqlite3.*", "execute")
        ir = matcher.to_ir()
        assert ir["receiverType"] == "sqlite3.*"

    def test_fully_qualified_type(self):
        """Test fully qualified type name."""
        matcher = calls_on("sqlite3.Cursor", "execute")
        ir = matcher.to_ir()
        assert ir["receiverType"] == "sqlite3.Cursor"

    def test_isinstance_check(self):
        """Test that calls_on returns TypeConstrainedCallMatcher."""
        matcher = calls_on("Cursor", "execute")
        assert isinstance(matcher, TypeConstrainedCallMatcher)

    def test_min_confidence_int(self):
        """Test that integer min_confidence is converted to float."""
        matcher = calls_on("Cursor", "execute", min_confidence=1)
        assert isinstance(matcher.min_confidence, float)
        assert matcher.min_confidence == 1.0

    def test_fallback_warn(self):
        """Test fallback='warn' mode."""
        matcher = calls_on("Cursor", "execute", fallback="warn")
        ir = matcher.to_ir()
        assert ir["fallbackMode"] == "warn"


class TestCallsReturning:
    """Test suite for calls_returning() matcher (deferred stub)."""

    def test_basic_ir(self):
        """Test calls_returning() produces correct IR structure."""
        matcher = calls_returning("str")
        ir = matcher.to_ir()

        assert ir["type"] == "return_type_call"
        assert ir["returnType"] == "str"
        assert ir["minConfidence"] == 0.5

    def test_custom_confidence(self):
        """Test calls_returning() with custom confidence."""
        matcher = calls_returning("List", min_confidence=0.8)
        ir = matcher.to_ir()
        assert ir["minConfidence"] == 0.8

    def test_validation_empty_type(self):
        """Test ValueError for empty return_type."""
        with pytest.raises(ValueError, match="return_type must be a non-empty string"):
            calls_returning("")

    def test_validation_none_type(self):
        """Test ValueError for None return_type."""
        with pytest.raises(ValueError, match="return_type must be a non-empty string"):
            calls_returning(None)

    def test_validation_bad_confidence(self):
        """Test ValueError for out-of-range confidence."""
        with pytest.raises(ValueError, match="min_confidence must be"):
            calls_returning("str", min_confidence=2.0)

    def test_validation_bad_confidence_negative(self):
        """Test ValueError for negative confidence."""
        with pytest.raises(ValueError, match="min_confidence must be"):
            calls_returning("str", min_confidence=-0.5)

    def test_repr(self):
        """Test string representation."""
        matcher = calls_returning("str")
        assert repr(matcher) == 'calls_returning("str")'

    def test_isinstance_check(self):
        """Test that calls_returning returns ReturnTypeCallMatcher."""
        matcher = calls_returning("str")
        assert isinstance(matcher, ReturnTypeCallMatcher)

    def test_min_confidence_int(self):
        """Test that integer min_confidence is converted to float."""
        matcher = calls_returning("str", min_confidence=1)
        assert isinstance(matcher.min_confidence, float)
        assert matcher.min_confidence == 1.0


class TestIRValidation:
    """Test IR validation for new types."""

    def test_validate_type_constrained_call(self):
        """Test validate_ir accepts valid type_constrained_call IR."""
        ir = calls_on("Cursor", "execute").to_ir()
        assert validate_ir(ir) is True

    def test_validate_type_constrained_call_missing_receiver(self):
        """Test validate_ir rejects missing receiverType."""
        ir = {"type": "type_constrained_call", "methodName": "execute",
              "minConfidence": 0.5, "fallbackMode": "name"}
        assert validate_ir(ir) is False

    def test_validate_type_constrained_call_empty_receiver(self):
        """Test validate_ir rejects empty receiverType."""
        ir = {"type": "type_constrained_call", "receiverType": "",
              "methodName": "execute", "minConfidence": 0.5, "fallbackMode": "name"}
        assert validate_ir(ir) is False

    def test_validate_type_constrained_call_bad_confidence(self):
        """Test validate_ir rejects out-of-range confidence."""
        ir = {"type": "type_constrained_call", "receiverType": "Cursor",
              "methodName": "execute", "minConfidence": 1.5, "fallbackMode": "name"}
        assert validate_ir(ir) is False

    def test_validate_type_constrained_call_bad_fallback(self):
        """Test validate_ir rejects invalid fallback mode."""
        ir = {"type": "type_constrained_call", "receiverType": "Cursor",
              "methodName": "execute", "minConfidence": 0.5, "fallbackMode": "invalid"}
        assert validate_ir(ir) is False

    def test_validate_return_type_call(self):
        """Test validate_ir accepts valid return_type_call IR."""
        ir = calls_returning("str").to_ir()
        assert validate_ir(ir) is True

    def test_validate_return_type_call_empty_type(self):
        """Test validate_ir rejects empty returnType."""
        ir = {"type": "return_type_call", "returnType": "", "minConfidence": 0.5}
        assert validate_ir(ir) is False

    def test_validate_return_type_call_bad_confidence(self):
        """Test validate_ir rejects out-of-range confidence."""
        ir = {"type": "return_type_call", "returnType": "str", "minConfidence": -0.1}
        assert validate_ir(ir) is False


class TestIRJSONMatchesGoStruct:
    """Verify JSON keys exactly match Go struct tags."""

    def test_type_constrained_call_keys(self):
        """Verify type_constrained_call IR keys match Go TypeConstrainedCallIR tags."""
        ir = calls_on("Cursor", "execute").to_ir()
        json_str = json.dumps(ir)
        parsed = json.loads(json_str)

        # These keys MUST match Go struct tags exactly:
        # TypeConstrainedCallIR in ir_types.go
        assert "type" in parsed
        assert "receiverType" in parsed
        assert "methodName" in parsed
        assert "minConfidence" in parsed
        assert "fallbackMode" in parsed
        # No extra keys
        assert set(parsed.keys()) == {
            "type", "receiverType", "methodName", "minConfidence", "fallbackMode"
        }

    def test_return_type_call_keys(self):
        """Verify return_type_call IR keys are well-formed."""
        ir = calls_returning("str").to_ir()
        json_str = json.dumps(ir)
        parsed = json.loads(json_str)

        assert "type" in parsed
        assert "returnType" in parsed
        assert "minConfidence" in parsed
        assert set(parsed.keys()) == {"type", "returnType", "minConfidence"}

    def test_json_round_trip_type_constrained(self):
        """Test JSON serialization round-trip for type_constrained_call."""
        ir = calls_on("Cursor", "execute", min_confidence=0.8, fallback="warn").to_ir()
        json_str = json.dumps(ir)
        parsed = json.loads(json_str)
        assert parsed == ir

    def test_json_round_trip_return_type(self):
        """Test JSON serialization round-trip for return_type_call."""
        ir = calls_returning("List", min_confidence=0.9).to_ir()
        json_str = json.dumps(ir)
        parsed = json.loads(json_str)
        assert parsed == ir


class TestRuleIntegration:
    """Test integration with @rule decorator."""

    def test_rule_with_calls_on(self):
        """Test that calls_on() works with @rule decorator."""
        from codepathfinder import rule

        @rule(id="TEST-001", severity="high")
        def detect_sql_injection():
            return calls_on("Cursor", "execute")

        result = detect_sql_injection.execute()
        assert result["rule"]["id"] == "TEST-001"
        assert result["matcher"]["type"] == "type_constrained_call"
        assert result["matcher"]["receiverType"] == "Cursor"

    def test_rule_with_calls_returning(self):
        """Test that calls_returning() works with @rule decorator."""
        from codepathfinder import rule

        @rule(id="TEST-002", severity="medium")
        def detect_return_type():
            return calls_returning("str")

        result = detect_return_type.execute()
        assert result["rule"]["id"] == "TEST-002"
        assert result["matcher"]["type"] == "return_type_call"
        assert result["matcher"]["returnType"] == "str"
