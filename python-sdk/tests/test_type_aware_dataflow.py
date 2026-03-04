"""Tests for type-aware dataflow: flows() with calls_on() and logic operators."""

import json

import pytest
from codepathfinder import flows, calls, calls_on, calls_returning, Or, And
from codepathfinder.dataflow import DataflowMatcher, _normalize_matchers
from codepathfinder.ir import IRType


class TestNormalizeMatchers:
    """Tests for _normalize_matchers() helper."""

    def test_single_call_matcher(self):
        """Single CallMatcher wraps in list."""
        m = _normalize_matchers(calls("eval"))
        assert len(m) == 1

    def test_single_calls_on(self):
        """Single TypeConstrainedCallMatcher wraps in list."""
        m = _normalize_matchers(calls_on("Cursor", "execute"))
        assert len(m) == 1

    def test_single_calls_returning(self):
        """Single ReturnTypeCallMatcher wraps in list."""
        m = _normalize_matchers(calls_returning("str"))
        assert len(m) == 1

    def test_list_passthrough(self):
        """List of matchers passes through unchanged."""
        m = _normalize_matchers([calls("a"), calls("b")])
        assert len(m) == 2

    def test_or_operator(self):
        """OrOperator wraps in list of 1."""
        m = _normalize_matchers(Or(calls("a"), calls("b")))
        assert len(m) == 1

    def test_and_operator(self):
        """AndOperator wraps in list of 1."""
        m = _normalize_matchers(And(calls("a"), calls("b")))
        assert len(m) == 1

    def test_none_returns_empty(self):
        """None returns empty list."""
        m = _normalize_matchers(None)
        assert m == []

    def test_invalid_raises_type_error(self):
        """Non-matcher raises TypeError."""
        with pytest.raises(TypeError, match="Expected a matcher"):
            _normalize_matchers(42)

    def test_string_raises_type_error(self):
        """String raises TypeError (not a matcher)."""
        with pytest.raises(TypeError, match="Expected a matcher"):
            _normalize_matchers("eval")


class TestFlowsWithCallsOn:
    """Tests for flows() accepting calls_on() matchers."""

    def test_calls_on_as_sink(self):
        """calls_on() works as single sink."""
        m = flows(
            from_sources=calls("request.GET"),
            to_sinks=calls_on("Cursor", "execute"),
        )
        ir = m.to_ir()
        assert len(ir["sinks"]) == 1
        assert ir["sinks"][0]["type"] == "type_constrained_call"
        assert ir["sinks"][0]["receiverType"] == "Cursor"

    def test_calls_on_as_source(self):
        """calls_on() works as single source."""
        m = flows(
            from_sources=calls_on("Request", "get_data"),
            to_sinks=calls("eval"),
        )
        ir = m.to_ir()
        assert len(ir["sources"]) == 1
        assert ir["sources"][0]["type"] == "type_constrained_call"

    def test_mixed_sinks(self):
        """Both calls() and calls_on() in same sinks list."""
        m = flows(
            from_sources=calls("request.GET"),
            to_sinks=[calls("eval"), calls_on("Cursor", "execute")],
        )
        ir = m.to_ir()
        assert len(ir["sinks"]) == 2
        types = {s["type"] for s in ir["sinks"]}
        assert types == {"call_matcher", "type_constrained_call"}

    def test_mixed_sources(self):
        """Both calls() and calls_on() in same sources list."""
        m = flows(
            from_sources=[calls("request.GET"), calls_on("Request", "get_data")],
            to_sinks=calls("eval"),
        )
        ir = m.to_ir()
        assert len(ir["sources"]) == 2
        types = {s["type"] for s in ir["sources"]}
        assert types == {"call_matcher", "type_constrained_call"}

    def test_calls_on_as_sanitizer(self):
        """calls_on() works as sanitizer."""
        m = flows(
            from_sources=calls("request.GET"),
            to_sinks=calls("eval"),
            sanitized_by=calls_on("Sanitizer", "clean"),
        )
        ir = m.to_ir()
        assert len(ir["sanitizers"]) == 1
        assert ir["sanitizers"][0]["type"] == "type_constrained_call"

    def test_backward_compat_call_matcher_only(self):
        """Existing call_matcher-only rules still work."""
        m = flows(
            from_sources=calls("request.GET"),
            to_sinks=calls("execute"),
        )
        ir = m.to_ir()
        assert ir["sinks"][0]["type"] == "call_matcher"

    def test_backward_compat_list_of_call_matchers(self):
        """Existing list-of-call-matchers rules still work."""
        m = flows(
            from_sources=[calls("a"), calls("b")],
            to_sinks=[calls("x"), calls("y")],
        )
        ir = m.to_ir()
        assert len(ir["sources"]) == 2
        assert len(ir["sinks"]) == 2


class TestFlowsWithLogicOperators:
    """Tests for flows() with Or/And as sources/sinks."""

    def test_or_as_sink(self):
        """Or() works as single sink wrapping multiple matchers."""
        m = flows(
            from_sources=calls("request.GET"),
            to_sinks=Or(calls_on("Cursor", "execute"), calls("eval")),
        )
        ir = m.to_ir()
        assert len(ir["sinks"]) == 1
        assert ir["sinks"][0]["type"] == "logic_or"
        assert len(ir["sinks"][0]["matchers"]) == 2

    def test_or_as_source(self):
        """Or() works as single source."""
        m = flows(
            from_sources=Or(calls("request.GET"), calls("request.POST")),
            to_sinks=calls("eval"),
        )
        ir = m.to_ir()
        assert len(ir["sources"]) == 1
        assert ir["sources"][0]["type"] == "logic_or"

    def test_and_as_sanitizer(self):
        """And() works as sanitizer."""
        m = flows(
            from_sources=calls("request.GET"),
            to_sinks=calls("eval"),
            sanitized_by=And(calls("escape"), calls("validate")),
        )
        ir = m.to_ir()
        assert len(ir["sanitizers"]) == 1
        assert ir["sanitizers"][0]["type"] == "logic_and"

    def test_or_with_mixed_matcher_types(self):
        """Or() containing both calls() and calls_on()."""
        m = flows(
            from_sources=calls("request.GET"),
            to_sinks=Or(calls("eval"), calls_on("Cursor", "execute")),
        )
        ir = m.to_ir()
        matchers = ir["sinks"][0]["matchers"]
        types = {m["type"] for m in matchers}
        assert types == {"call_matcher", "type_constrained_call"}


class TestFlowsIRRoundTrip:
    """Test JSON round-trip for polymorphic dataflow IR."""

    def test_mixed_types_json_round_trip(self):
        """Mixed sink types survive JSON round-trip."""
        m = flows(
            from_sources=calls("request.GET"),
            to_sinks=[calls("eval"), calls_on("Cursor", "execute")],
        )
        ir = m.to_ir()
        json_str = json.dumps(ir)
        parsed = json.loads(json_str)
        assert parsed == ir

    def test_or_sink_json_round_trip(self):
        """Or() sink survives JSON round-trip."""
        m = flows(
            from_sources=calls("request.GET"),
            to_sinks=Or(calls_on("Cursor", "execute"), calls("eval")),
        )
        ir = m.to_ir()
        json_str = json.dumps(ir)
        parsed = json.loads(json_str)
        assert parsed == ir

    def test_full_polymorphic_ir_structure(self):
        """Full IR with mixed types has correct structure."""
        m = flows(
            from_sources=Or(calls("request.GET"), calls_on("Request", "get_data")),
            to_sinks=[calls("eval"), calls_on("Cursor", "execute")],
            sanitized_by=calls("escape"),
            scope="global",
        )
        ir = m.to_ir()
        assert ir["type"] == "dataflow"
        assert ir["scope"] == "global"
        assert len(ir["sources"]) == 1  # Or wraps into 1
        assert len(ir["sinks"]) == 2
        assert len(ir["sanitizers"]) == 1
