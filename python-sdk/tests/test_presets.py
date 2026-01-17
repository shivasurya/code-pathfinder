"""Tests for PropagationPresets."""

from codepathfinder.presets import PropagationPresets
from codepathfinder.propagation import PropagationType, PropagationPrimitive


class TestPropagationPresets:
    """Test propagation preset bundles."""

    def test_minimal_returns_list(self):
        """Test minimal preset returns list of primitives."""
        prims = PropagationPresets.minimal()
        assert isinstance(prims, list)
        assert all(isinstance(p, PropagationPrimitive) for p in prims)

    def test_minimal_count_and_types(self):
        """Test minimal preset has correct primitives."""
        prims = PropagationPresets.minimal()
        assert len(prims) == 2
        assert prims[0].type == PropagationType.ASSIGNMENT
        assert prims[1].type == PropagationType.FUNCTION_ARGS

    def test_standard_returns_list(self):
        """Test standard preset returns list of primitives."""
        prims = PropagationPresets.standard()
        assert isinstance(prims, list)
        assert all(isinstance(p, PropagationPrimitive) for p in prims)

    def test_standard_count_and_types(self):
        """Test standard preset (recommended) has all Phase 1+2 primitives."""
        prims = PropagationPresets.standard()
        assert len(prims) == 5
        types = [p.type for p in prims]
        assert PropagationType.ASSIGNMENT in types
        assert PropagationType.FUNCTION_ARGS in types
        assert PropagationType.FUNCTION_RETURNS in types
        assert PropagationType.STRING_CONCAT in types
        assert PropagationType.STRING_FORMAT in types

    def test_comprehensive_equals_standard_for_mvp(self):
        """Test comprehensive preset (MVP all) is same as standard for MVP."""
        comp_prims = PropagationPresets.comprehensive()
        std_prims = PropagationPresets.standard()
        assert len(comp_prims) == len(std_prims)
        assert len(comp_prims) == 5  # For MVP, same as standard

    def test_exhaustive_equals_comprehensive_for_mvp(self):
        """Test exhaustive preset (future: all phases) is same as comprehensive for MVP."""
        exh_prims = PropagationPresets.exhaustive()
        comp_prims = PropagationPresets.comprehensive()
        assert len(exh_prims) == len(comp_prims)
        assert len(exh_prims) >= 5  # For MVP, same as comprehensive

    def test_minimal_serializes_to_ir(self):
        """Test minimal preset primitives can serialize to IR."""
        prims = PropagationPresets.minimal()
        ir_list = [p.to_ir() for p in prims]
        assert len(ir_list) == 2
        assert ir_list[0]["type"] == "assignment"
        assert ir_list[1]["type"] == "function_args"

    def test_standard_serializes_to_ir(self):
        """Test standard preset primitives can serialize to IR."""
        prims = PropagationPresets.standard()
        ir_list = [p.to_ir() for p in prims]
        assert len(ir_list) == 5
        assert all("type" in ir for ir in ir_list)
        assert all("metadata" in ir for ir in ir_list)


class TestPresetOrdering:
    """Test that presets maintain consistent ordering."""

    def test_minimal_order(self):
        """Test minimal preset has consistent order."""
        prims = PropagationPresets.minimal()
        types = [p.type.value for p in prims]
        assert types == ["assignment", "function_args"]

    def test_standard_order(self):
        """Test standard preset has consistent order."""
        prims = PropagationPresets.standard()
        types = [p.type.value for p in prims]
        assert types == [
            "assignment",
            "function_args",
            "function_returns",
            "string_concat",
            "string_format",
        ]


class TestPresetUsage:
    """Test realistic usage patterns with presets."""

    def test_preset_can_be_used_with_flows(self):
        """Test presets can be passed to flows() propagates_through parameter."""
        from codepathfinder import flows, calls

        matcher = flows(
            from_sources=calls("request.GET"),
            to_sinks=calls("eval"),
            propagates_through=PropagationPresets.minimal(),
        )
        assert len(matcher.propagates_through) == 2

    def test_preset_standard_with_flows(self):
        """Test standard preset with flows()."""
        from codepathfinder import flows, calls

        matcher = flows(
            from_sources=calls("request.GET"),
            to_sinks=calls("execute"),
            propagates_through=PropagationPresets.standard(),
        )
        assert len(matcher.propagates_through) == 5
