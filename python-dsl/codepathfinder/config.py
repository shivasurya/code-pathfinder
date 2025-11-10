"""
Global configuration for codepathfinder DSL.

Allows setting default propagation, scope, etc.
"""

from typing import List, Optional
from .propagation import PropagationPrimitive


class PathfinderConfig:
    """Singleton configuration for codepathfinder."""

    _instance: Optional["PathfinderConfig"] = None
    _default_propagation: List[PropagationPrimitive] = []
    _default_scope: str = "global"

    def __new__(cls):
        if cls._instance is None:
            cls._instance = super().__new__(cls)
        return cls._instance

    @property
    def default_propagation(self) -> List[PropagationPrimitive]:
        """Get default propagation primitives."""
        return self._default_propagation

    @default_propagation.setter
    def default_propagation(self, value: List[PropagationPrimitive]):
        """Set default propagation primitives."""
        self._default_propagation = value

    @property
    def default_scope(self) -> str:
        """Get default scope."""
        return self._default_scope

    @default_scope.setter
    def default_scope(self, value: str):
        """Set default scope."""
        if value not in ["local", "global"]:
            raise ValueError(f"scope must be 'local' or 'global', got '{value}'")
        self._default_scope = value


# Global config instance
_config = PathfinderConfig()


def set_default_propagation(primitives: List[PropagationPrimitive]) -> None:
    """
    Set global default propagation primitives.

    All flows() calls without explicit propagates_through will use this default.

    Args:
        primitives: List of PropagationPrimitive objects

    Example:
        set_default_propagation(PropagationPresets.standard())

        # Now all flows() without propagates_through use standard()
        flows(
            from_sources=calls("request.GET"),
            to_sinks=calls("eval"),
            # propagates_through defaults to standard()
        )
    """
    _config.default_propagation = primitives


def set_default_scope(scope: str) -> None:
    """
    Set global default scope.

    Args:
        scope: "local" or "global"

    Example:
        set_default_scope("local")
    """
    _config.default_scope = scope


def get_default_propagation() -> List[PropagationPrimitive]:
    """Get global default propagation primitives."""
    return _config.default_propagation


def get_default_scope() -> str:
    """Get global default scope."""
    return _config.default_scope
