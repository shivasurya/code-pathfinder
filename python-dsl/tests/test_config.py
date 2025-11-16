"""Tests for global configuration."""

import pytest
from codepathfinder import (
    set_default_propagation,
    set_default_scope,
    PropagationPresets,
    flows,
    calls,
    propagates,
)
from codepathfinder.config import (
    get_default_propagation,
    get_default_scope,
    PathfinderConfig,
)


class TestPathfinderConfig:
    """Test PathfinderConfig singleton."""

    def test_singleton_instance(self):
        """Test PathfinderConfig is a singleton."""
        config1 = PathfinderConfig()
        config2 = PathfinderConfig()
        assert config1 is config2

    def test_default_propagation_property(self):
        """Test default_propagation property getter/setter."""
        config = PathfinderConfig()
        config.default_propagation = PropagationPresets.minimal()
        assert len(config.default_propagation) == 2

    def test_default_scope_property(self):
        """Test default_scope property getter/setter."""
        config = PathfinderConfig()
        config.default_scope = "local"
        assert config.default_scope == "local"

    def test_invalid_scope_raises(self):
        """Test setting invalid scope raises ValueError."""
        config = PathfinderConfig()
        with pytest.raises(ValueError, match="scope must be"):
            config.default_scope = "invalid"


class TestGlobalConfig:
    """Test global configuration functions."""

    def test_set_default_propagation(self):
        """Test setting default propagation."""
        set_default_propagation(PropagationPresets.minimal())
        result = get_default_propagation()
        assert len(result) == 2

    def test_set_default_propagation_standard(self):
        """Test setting standard propagation."""
        set_default_propagation(PropagationPresets.standard())
        result = get_default_propagation()
        assert len(result) == 5

    def test_set_default_propagation_custom(self):
        """Test setting custom propagation list."""
        custom = [propagates.assignment(), propagates.string_concat()]
        set_default_propagation(custom)
        result = get_default_propagation()
        assert len(result) == 2

    def test_set_default_scope_local(self):
        """Test setting default scope to local."""
        set_default_scope("local")
        assert get_default_scope() == "local"

    def test_set_default_scope_global(self):
        """Test setting default scope to global."""
        set_default_scope("global")
        assert get_default_scope() == "global"

    def test_invalid_scope_function_raises(self):
        """Test set_default_scope with invalid scope raises ValueError."""
        with pytest.raises(ValueError, match="scope must be"):
            set_default_scope("invalid")


class TestFlowsWithDefaults:
    """Test flows() uses global defaults when not specified."""

    def test_flows_uses_default_propagation(self):
        """Test flows() uses global default propagation when not specified."""
        # Set default
        set_default_propagation(PropagationPresets.minimal())

        # Create matcher without specifying propagates_through
        matcher = flows(
            from_sources=calls("request.GET"),
            to_sinks=calls("eval"),
            # propagates_through NOT specified
        )

        assert len(matcher.propagates_through) == 2

    def test_flows_uses_default_scope(self):
        """Test flows() uses global default scope when not specified."""
        # Set default
        set_default_scope("local")

        # Create matcher without specifying scope
        matcher = flows(
            from_sources=calls("request.GET"),
            to_sinks=calls("eval"),
            propagates_through=[],
            # scope NOT specified
        )

        assert matcher.scope == "local"

    def test_flows_override_default_propagation(self):
        """Test flows() can override default propagation."""
        # Set default to minimal
        set_default_propagation(PropagationPresets.minimal())

        # Override with standard
        matcher = flows(
            from_sources=calls("request.GET"),
            to_sinks=calls("eval"),
            propagates_through=PropagationPresets.standard(),  # OVERRIDE
        )

        assert len(matcher.propagates_through) == 5  # Not 2

    def test_flows_override_default_scope(self):
        """Test flows() can override default scope."""
        # Set default to local
        set_default_scope("local")

        # Override with global
        matcher = flows(
            from_sources=calls("request.GET"),
            to_sinks=calls("eval"),
            propagates_through=[],
            scope="global",  # OVERRIDE
        )

        assert matcher.scope == "global"  # Not local

    def test_flows_with_empty_default_propagation(self):
        """Test flows() with empty default propagation."""
        # Set default to empty list
        set_default_propagation([])

        matcher = flows(
            from_sources=calls("request.GET"),
            to_sinks=calls("eval"),
            # propagates_through NOT specified (uses empty default)
        )

        assert matcher.propagates_through == []

    def test_flows_explicit_empty_overrides_default(self):
        """Test flows() with explicit empty list overrides default."""
        # Set default to standard
        set_default_propagation(PropagationPresets.standard())

        # Explicitly pass empty list
        matcher = flows(
            from_sources=calls("request.GET"),
            to_sinks=calls("eval"),
            propagates_through=[],  # EXPLICIT empty
        )

        assert matcher.propagates_through == []  # Not standard


class TestDefaultsIntegration:
    """Integration tests for global defaults."""

    def test_complete_default_workflow(self):
        """Test complete workflow with defaults."""
        # Setup defaults
        set_default_propagation(PropagationPresets.standard())
        set_default_scope("global")

        # Create matcher using defaults
        matcher = flows(
            from_sources=calls("request.GET"),
            to_sinks=calls("execute"),
        )

        assert len(matcher.propagates_through) == 5
        assert matcher.scope == "global"

    def test_partial_override_workflow(self):
        """Test workflow with partial overrides."""
        # Setup defaults
        set_default_propagation(PropagationPresets.standard())
        set_default_scope("global")

        # Override only scope
        matcher = flows(
            from_sources=calls("request.GET"),
            to_sinks=calls("execute"),
            # propagates_through uses default (standard)
            scope="local",  # override
        )

        assert len(matcher.propagates_through) == 5  # from default
        assert matcher.scope == "local"  # overridden
