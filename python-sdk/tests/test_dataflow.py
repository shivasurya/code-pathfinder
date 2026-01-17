"""
Tests for dataflow matcher and flows() function.
"""

import pytest
from codepathfinder.dataflow import DataflowMatcher, flows
from codepathfinder.matchers import calls, variable
from codepathfinder.propagation import propagates
from codepathfinder.ir import IRType


class TestDataflowMatcherInit:
    """Tests for DataflowMatcher initialization."""

    def test_create_with_single_source_and_sink(self):
        """Can create matcher with single source and sink."""
        matcher = DataflowMatcher(
            from_sources=calls("request.GET"),
            to_sinks=calls("execute"),
            propagates_through=[],  # Explicit empty
            scope="global",  # Explicit scope
        )
        assert len(matcher.sources) == 1
        assert len(matcher.sinks) == 1
        assert matcher.sanitizers == []
        assert matcher.propagates_through == []
        assert matcher.scope == "global"

    def test_create_with_multiple_sources(self):
        """Can create matcher with multiple sources."""
        matcher = DataflowMatcher(
            from_sources=[calls("request.GET"), calls("request.POST")],
            to_sinks=calls("execute"),
        )
        assert len(matcher.sources) == 2

    def test_create_with_multiple_sinks(self):
        """Can create matcher with multiple sinks."""
        matcher = DataflowMatcher(
            from_sources=calls("request.GET"),
            to_sinks=[calls("execute"), calls("executemany")],
        )
        assert len(matcher.sinks) == 2

    def test_create_with_sanitizers(self):
        """Can create matcher with sanitizers."""
        matcher = DataflowMatcher(
            from_sources=calls("request.GET"),
            to_sinks=calls("execute"),
            sanitized_by=calls("quote_sql"),
        )
        assert len(matcher.sanitizers) == 1

    def test_create_with_multiple_sanitizers(self):
        """Can create matcher with multiple sanitizers."""
        matcher = DataflowMatcher(
            from_sources=calls("request.GET"),
            to_sinks=calls("execute"),
            sanitized_by=[calls("quote_sql"), calls("escape_sql")],
        )
        assert len(matcher.sanitizers) == 2

    def test_create_with_propagation(self):
        """Can create matcher with propagation primitives."""
        matcher = DataflowMatcher(
            from_sources=calls("request.GET"),
            to_sinks=calls("execute"),
            propagates_through=[propagates.assignment()],
        )
        assert len(matcher.propagates_through) == 1

    def test_create_with_multiple_propagation(self):
        """Can create matcher with multiple propagation primitives."""
        matcher = DataflowMatcher(
            from_sources=calls("request.GET"),
            to_sinks=calls("execute"),
            propagates_through=[
                propagates.assignment(),
                propagates.function_args(),
                propagates.function_returns(),
            ],
        )
        assert len(matcher.propagates_through) == 3

    def test_create_with_local_scope(self):
        """Can create matcher with local scope."""
        matcher = DataflowMatcher(
            from_sources=calls("request.GET"),
            to_sinks=calls("execute"),
            scope="local",
        )
        assert matcher.scope == "local"

    def test_empty_sources_raises_error(self):
        """Empty sources raises ValueError."""
        with pytest.raises(ValueError, match="requires at least one source"):
            DataflowMatcher(from_sources=[], to_sinks=calls("execute"))

    def test_empty_sinks_raises_error(self):
        """Empty sinks raises ValueError."""
        with pytest.raises(ValueError, match="requires at least one sink"):
            DataflowMatcher(from_sources=calls("request.GET"), to_sinks=[])

    def test_invalid_scope_raises_error(self):
        """Invalid scope raises ValueError."""
        with pytest.raises(ValueError, match="scope must be"):
            DataflowMatcher(
                from_sources=calls("request.GET"),
                to_sinks=calls("execute"),
                scope="invalid",
            )


class TestDataflowMatcherToIR:
    """Tests for DataflowMatcher.to_ir() serialization."""

    def test_minimal_ir(self):
        """Minimal matcher serializes correctly."""
        matcher = DataflowMatcher(
            from_sources=calls("source"),
            to_sinks=calls("sink"),
            propagates_through=[],  # Explicit empty
            scope="global",  # Explicit scope
        )
        ir = matcher.to_ir()
        assert ir["type"] == IRType.DATAFLOW.value
        assert len(ir["sources"]) == 1
        assert len(ir["sinks"]) == 1
        assert ir["sanitizers"] == []
        assert ir["propagation"] == []
        assert ir["scope"] == "global"

    def test_full_ir_with_all_fields(self):
        """Full matcher with all fields serializes correctly."""
        matcher = DataflowMatcher(
            from_sources=[calls("request.GET"), calls("request.POST")],
            to_sinks=[calls("execute"), calls("executemany")],
            sanitized_by=[calls("quote_sql")],
            propagates_through=[
                propagates.assignment(),
                propagates.function_args(),
            ],
            scope="local",
        )
        ir = matcher.to_ir()
        assert ir["type"] == IRType.DATAFLOW.value
        assert len(ir["sources"]) == 2
        assert len(ir["sinks"]) == 2
        assert len(ir["sanitizers"]) == 1
        assert len(ir["propagation"]) == 2
        assert ir["scope"] == "local"

    def test_sources_ir_structure(self):
        """Sources serialize to correct IR structure."""
        matcher = DataflowMatcher(
            from_sources=calls("request.GET"),
            to_sinks=calls("execute"),
        )
        ir = matcher.to_ir()
        source_ir = ir["sources"][0]
        assert source_ir["type"] == "call_matcher"
        assert "request.GET" in source_ir["patterns"]

    def test_sinks_ir_structure(self):
        """Sinks serialize to correct IR structure."""
        matcher = DataflowMatcher(
            from_sources=calls("request.GET"),
            to_sinks=calls("execute"),
        )
        ir = matcher.to_ir()
        sink_ir = ir["sinks"][0]
        assert sink_ir["type"] == "call_matcher"
        assert "execute" in sink_ir["patterns"]

    def test_sanitizers_ir_structure(self):
        """Sanitizers serialize to correct IR structure."""
        matcher = DataflowMatcher(
            from_sources=calls("request.GET"),
            to_sinks=calls("execute"),
            sanitized_by=calls("quote_sql"),
        )
        ir = matcher.to_ir()
        sanitizer_ir = ir["sanitizers"][0]
        assert sanitizer_ir["type"] == "call_matcher"
        assert "quote_sql" in sanitizer_ir["patterns"]

    def test_propagation_ir_structure(self):
        """Propagation primitives serialize to correct IR structure."""
        matcher = DataflowMatcher(
            from_sources=calls("request.GET"),
            to_sinks=calls("execute"),
            propagates_through=[propagates.assignment()],
        )
        ir = matcher.to_ir()
        prop_ir = ir["propagation"][0]
        assert prop_ir["type"] == "assignment"
        assert prop_ir["metadata"] == {}


class TestDataflowMatcherRepr:
    """Tests for DataflowMatcher.__repr__()."""

    def test_repr_format(self):
        """__repr__ returns readable string."""
        matcher = DataflowMatcher(
            from_sources=calls("request.GET"),
            to_sinks=calls("execute"),
            propagates_through=[propagates.assignment()],
        )
        repr_str = repr(matcher)
        assert "flows" in repr_str
        assert "sources=1" in repr_str
        assert "sinks=1" in repr_str
        assert "propagation=1" in repr_str
        assert "scope='global'" in repr_str

    def test_repr_counts_multiple(self):
        """__repr__ counts multiple sources/sinks/propagation."""
        matcher = DataflowMatcher(
            from_sources=[calls("a"), calls("b")],
            to_sinks=[calls("x"), calls("y"), calls("z")],
            propagates_through=[propagates.assignment(), propagates.function_args()],
        )
        repr_str = repr(matcher)
        assert "sources=2" in repr_str
        assert "sinks=3" in repr_str
        assert "propagation=2" in repr_str


class TestFlowsFunction:
    """Tests for flows() public API function."""

    def test_flows_returns_dataflow_matcher(self):
        """flows() returns DataflowMatcher instance."""
        matcher = flows(
            from_sources=calls("request.GET"),
            to_sinks=calls("execute"),
        )
        assert isinstance(matcher, DataflowMatcher)

    def test_flows_with_all_parameters(self):
        """flows() accepts all parameters."""
        matcher = flows(
            from_sources=calls("request.GET"),
            to_sinks=calls("execute"),
            sanitized_by=calls("quote_sql"),
            propagates_through=[propagates.assignment()],
            scope="local",
        )
        assert len(matcher.sources) == 1
        assert len(matcher.sinks) == 1
        assert len(matcher.sanitizers) == 1
        assert len(matcher.propagates_through) == 1
        assert matcher.scope == "local"

    def test_flows_default_scope_is_global(self):
        """flows() defaults to global scope."""
        matcher = flows(
            from_sources=calls("request.GET"),
            to_sinks=calls("execute"),
        )
        assert matcher.scope == "global"

    def test_flows_default_propagation_uses_global_config(self):
        """flows() uses global default propagation when not specified."""
        # This test now reflects PR #4 behavior
        from codepathfinder import set_default_propagation

        # Set a known default
        set_default_propagation([])

        matcher = flows(
            from_sources=calls("request.GET"),
            to_sinks=calls("execute"),
        )
        assert matcher.propagates_through == []

    def test_flows_default_sanitizers_is_empty(self):
        """flows() defaults to empty sanitizers list."""
        matcher = flows(
            from_sources=calls("request.GET"),
            to_sinks=calls("execute"),
        )
        assert matcher.sanitizers == []


class TestDataflowIntegration:
    """Integration tests for realistic OWASP Top 10 patterns."""

    def test_sql_injection_pattern(self):
        """SQL injection pattern with typical configuration."""
        matcher = flows(
            from_sources=calls("request.GET", "request.POST"),
            to_sinks=calls("execute", "executemany"),
            sanitized_by=calls("quote_sql"),
            propagates_through=[
                propagates.assignment(),
                propagates.function_args(),
            ],
            scope="global",
        )
        ir = matcher.to_ir()
        assert ir["type"] == "dataflow"
        assert len(ir["sources"]) == 1  # Single calls() with 2 patterns
        assert len(ir["sinks"]) == 1  # Single calls() with 2 patterns
        assert len(ir["propagation"]) == 2

    def test_command_injection_pattern(self):
        """Command injection pattern with typical configuration."""
        matcher = flows(
            from_sources=calls("request.POST"),
            to_sinks=calls("os.system", "subprocess.call"),
            sanitized_by=calls("shlex.quote"),
            propagates_through=[
                propagates.assignment(),
                propagates.function_args(),
                propagates.function_returns(),
            ],
            scope="global",
        )
        ir = matcher.to_ir()
        assert ir["type"] == "dataflow"
        assert len(ir["propagation"]) == 3

    def test_path_traversal_pattern(self):
        """Path traversal pattern with typical configuration."""
        matcher = flows(
            from_sources=calls("request.GET"),
            to_sinks=calls("open", "os.path.join"),
            sanitized_by=calls("os.path.abspath"),
            propagates_through=[propagates.assignment()],
            scope="local",
        )
        ir = matcher.to_ir()
        assert ir["scope"] == "local"
        assert len(ir["propagation"]) == 1

    def test_ssrf_pattern(self):
        """SSRF pattern with typical configuration."""
        matcher = flows(
            from_sources=calls("request.GET", "request.POST"),
            to_sinks=calls("requests.get", "urllib.request.urlopen"),
            propagates_through=[
                propagates.assignment(),
                propagates.function_args(),
            ],
            scope="global",
        )
        ir = matcher.to_ir()
        assert len(ir["sanitizers"]) == 0  # No sanitizers for SSRF

    def test_insecure_deserialization_pattern(self):
        """Insecure deserialization pattern."""
        matcher = flows(
            from_sources=calls("request.POST"),
            to_sinks=calls("pickle.loads", "yaml.load"),
            propagates_through=[propagates.assignment()],
            scope="local",
        )
        ir = matcher.to_ir()
        assert ir["type"] == "dataflow"


class TestDataflowEdgeCases:
    """Edge cases and error handling tests."""

    def test_mixed_matcher_types_sources(self):
        """Can mix CallMatcher and VariableMatcher as sources."""
        matcher = flows(
            from_sources=[calls("request.GET"), variable("user_*")],
            to_sinks=calls("execute"),
        )
        assert len(matcher.sources) == 2

    def test_single_propagation_primitive(self):
        """Can specify single propagation primitive."""
        matcher = flows(
            from_sources=calls("source"),
            to_sinks=calls("sink"),
            propagates_through=[propagates.assignment()],
        )
        assert len(matcher.propagates_through) == 1

    def test_no_propagation_valid(self):
        """No propagation is valid (explicit choice)."""
        matcher = flows(
            from_sources=calls("source"),
            to_sinks=calls("sink"),
            propagates_through=[],
        )
        assert matcher.propagates_through == []

    def test_global_scope_string_validation(self):
        """'global' scope string is accepted."""
        matcher = flows(
            from_sources=calls("source"),
            to_sinks=calls("sink"),
            scope="global",
        )
        assert matcher.scope == "global"

    def test_local_scope_string_validation(self):
        """'local' scope string is accepted."""
        matcher = flows(
            from_sources=calls("source"),
            to_sinks=calls("sink"),
            scope="local",
        )
        assert matcher.scope == "local"
