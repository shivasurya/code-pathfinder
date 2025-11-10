"""Tests for Phase 2 propagation primitives."""

from codepathfinder.propagation import propagates, PropagationType, PropagationPrimitive


class TestPhase2Primitives:
    """Test Phase 2 string operation primitives."""

    def test_string_concat_returns_primitive(self):
        """Test propagates.string_concat() returns PropagationPrimitive."""
        prim = propagates.string_concat()
        assert isinstance(prim, PropagationPrimitive)
        assert prim.type == PropagationType.STRING_CONCAT

    def test_string_concat_ir(self):
        """Test propagates.string_concat() serializes correctly."""
        prim = propagates.string_concat()
        ir = prim.to_ir()
        assert ir["type"] == "string_concat"
        assert ir["metadata"] == {}

    def test_string_format_returns_primitive(self):
        """Test propagates.string_format() returns PropagationPrimitive."""
        prim = propagates.string_format()
        assert isinstance(prim, PropagationPrimitive)
        assert prim.type == PropagationType.STRING_FORMAT

    def test_string_format_ir(self):
        """Test propagates.string_format() serializes correctly."""
        prim = propagates.string_format()
        ir = prim.to_ir()
        assert ir["type"] == "string_format"
        assert ir["metadata"] == {}

    def test_string_concat_repr(self):
        """Test string_concat __repr__."""
        prim = propagates.string_concat()
        assert repr(prim) == "propagates.string_concat()"

    def test_string_format_repr(self):
        """Test string_format __repr__."""
        prim = propagates.string_format()
        assert repr(prim) == "propagates.string_format()"


class TestPhase2Integration:
    """Integration tests for Phase 2 primitives."""

    def test_all_phase2_primitives(self):
        """Test all Phase 2 primitives can be used together."""
        prims = [
            propagates.string_concat(),
            propagates.string_format(),
        ]
        assert len(prims) == 2
        types = [p.type for p in prims]
        assert PropagationType.STRING_CONCAT in types
        assert PropagationType.STRING_FORMAT in types

    def test_phase1_and_phase2_together(self):
        """Test Phase 1 and Phase 2 primitives work together."""
        prims = [
            propagates.assignment(),
            propagates.function_args(),
            propagates.function_returns(),
            propagates.string_concat(),
            propagates.string_format(),
        ]
        assert len(prims) == 5
        # All primitives should serialize to IR
        ir_list = [p.to_ir() for p in prims]
        assert len(ir_list) == 5
