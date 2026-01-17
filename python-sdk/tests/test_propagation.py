"""
Tests for taint propagation primitives.
"""

from codepathfinder.propagation import (
    PropagationType,
    PropagationPrimitive,
    propagates,
    create_propagation_list,
)


class TestPropagationType:
    """Tests for PropagationType enum."""

    def test_phase1_types_exist(self):
        """Phase 1 propagation types are defined."""
        assert PropagationType.ASSIGNMENT.value == "assignment"
        assert PropagationType.FUNCTION_ARGS.value == "function_args"
        assert PropagationType.FUNCTION_RETURNS.value == "function_returns"

    def test_phase2_types_exist(self):
        """Phase 2 propagation types are defined (not implemented yet)."""
        assert PropagationType.STRING_CONCAT.value == "string_concat"
        assert PropagationType.STRING_FORMAT.value == "string_format"

    def test_all_enum_values_unique(self):
        """All enum values are unique."""
        values = [t.value for t in PropagationType]
        assert len(values) == len(set(values))


class TestPropagationPrimitive:
    """Tests for PropagationPrimitive base class."""

    def test_create_primitive_without_metadata(self):
        """Can create primitive without metadata."""
        prim = PropagationPrimitive(PropagationType.ASSIGNMENT)
        assert prim.type == PropagationType.ASSIGNMENT
        assert prim.metadata == {}

    def test_create_primitive_with_metadata(self):
        """Can create primitive with metadata."""
        metadata = {"key": "value"}
        prim = PropagationPrimitive(PropagationType.ASSIGNMENT, metadata)
        assert prim.type == PropagationType.ASSIGNMENT
        assert prim.metadata == metadata

    def test_to_ir_without_metadata(self):
        """to_ir() returns correct JSON IR without metadata."""
        prim = PropagationPrimitive(PropagationType.ASSIGNMENT)
        ir = prim.to_ir()
        assert ir == {"type": "assignment", "metadata": {}}

    def test_to_ir_with_metadata(self):
        """to_ir() returns correct JSON IR with metadata."""
        metadata = {"foo": "bar", "baz": 42}
        prim = PropagationPrimitive(PropagationType.FUNCTION_ARGS, metadata)
        ir = prim.to_ir()
        assert ir == {"type": "function_args", "metadata": metadata}

    def test_repr(self):
        """__repr__ returns readable string."""
        prim = PropagationPrimitive(PropagationType.ASSIGNMENT)
        assert repr(prim) == "propagates.assignment()"


class TestPropagatesNamespace:
    """Tests for propagates namespace (Phase 1 methods)."""

    def test_assignment_returns_primitive(self):
        """propagates.assignment() returns PropagationPrimitive."""
        prim = propagates.assignment()
        assert isinstance(prim, PropagationPrimitive)
        assert prim.type == PropagationType.ASSIGNMENT

    def test_assignment_ir(self):
        """propagates.assignment() serializes correctly."""
        prim = propagates.assignment()
        ir = prim.to_ir()
        assert ir == {"type": "assignment", "metadata": {}}

    def test_function_args_returns_primitive(self):
        """propagates.function_args() returns PropagationPrimitive."""
        prim = propagates.function_args()
        assert isinstance(prim, PropagationPrimitive)
        assert prim.type == PropagationType.FUNCTION_ARGS

    def test_function_args_ir(self):
        """propagates.function_args() serializes correctly."""
        prim = propagates.function_args()
        ir = prim.to_ir()
        assert ir == {"type": "function_args", "metadata": {}}

    def test_function_returns_returns_primitive(self):
        """propagates.function_returns() returns PropagationPrimitive."""
        prim = propagates.function_returns()
        assert isinstance(prim, PropagationPrimitive)
        assert prim.type == PropagationType.FUNCTION_RETURNS

    def test_function_returns_ir(self):
        """propagates.function_returns() serializes correctly."""
        prim = propagates.function_returns()
        ir = prim.to_ir()
        assert ir == {"type": "function_returns", "metadata": {}}


class TestCreatePropagationList:
    """Tests for create_propagation_list helper."""

    def test_empty_list(self):
        """Empty list returns empty JSON IR list."""
        ir_list = create_propagation_list([])
        assert ir_list == []

    def test_single_primitive(self):
        """Single primitive returns single JSON IR dict."""
        prims = [propagates.assignment()]
        ir_list = create_propagation_list(prims)
        assert len(ir_list) == 1
        assert ir_list[0] == {"type": "assignment", "metadata": {}}

    def test_multiple_primitives(self):
        """Multiple primitives return multiple JSON IR dicts."""
        prims = [
            propagates.assignment(),
            propagates.function_args(),
            propagates.function_returns(),
        ]
        ir_list = create_propagation_list(prims)
        assert len(ir_list) == 3
        assert ir_list[0] == {"type": "assignment", "metadata": {}}
        assert ir_list[1] == {"type": "function_args", "metadata": {}}
        assert ir_list[2] == {"type": "function_returns", "metadata": {}}

    def test_preserves_order(self):
        """Primitive order is preserved in JSON IR."""
        prims = [
            propagates.function_returns(),
            propagates.assignment(),
            propagates.function_args(),
        ]
        ir_list = create_propagation_list(prims)
        assert ir_list[0]["type"] == "function_returns"
        assert ir_list[1]["type"] == "assignment"
        assert ir_list[2]["type"] == "function_args"


class TestPropagationIntegration:
    """Integration tests for propagation primitives."""

    def test_typical_sql_injection_propagation(self):
        """Typical SQL injection uses assignment + function_args."""
        prims = [propagates.assignment(), propagates.function_args()]
        ir_list = create_propagation_list(prims)
        assert len(ir_list) == 2
        assert all(isinstance(ir, dict) for ir in ir_list)

    def test_typical_command_injection_propagation(self):
        """Typical command injection uses all three Phase 1 primitives."""
        prims = [
            propagates.assignment(),
            propagates.function_args(),
            propagates.function_returns(),
        ]
        ir_list = create_propagation_list(prims)
        assert len(ir_list) == 3

    def test_minimal_propagation(self):
        """Can use just assignment for intra-procedural analysis."""
        prims = [propagates.assignment()]
        ir_list = create_propagation_list(prims)
        assert len(ir_list) == 1
        assert ir_list[0]["type"] == "assignment"
