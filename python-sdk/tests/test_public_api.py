"""Guards against accidental public-API breakage.

The set of names re-exported from `codepathfinder` is what downstream users
depend on. This test pins that set so any addition or removal shows up as a
deliberate change in code review.
"""

import codepathfinder

# Snapshot of the supported public API at this point in time. Adding a new
# public symbol? Append it here in the same PR. Removing one? That's a
# breaking change that needs a major-version bump and a deprecation cycle.
EXPECTED_PUBLIC_NAMES = frozenset(
    {
        "__version__",
        "attribute",
        "calls",
        "variable",
        "rule",
        "flows",
        "propagates",
        "PropagationPresets",
        "set_default_propagation",
        "set_default_scope",
        "And",
        "Or",
        "Not",
        "QueryType",
        "lt",
        "gt",
        "lte",
        "gte",
        "regex",
        "missing",
    }
)


def test_all_matches_snapshot() -> None:
    """`__all__` must match the pinned snapshot exactly."""
    assert frozenset(codepathfinder.__all__) == EXPECTED_PUBLIC_NAMES


def test_every_name_in_all_is_importable() -> None:
    """Every name in `__all__` must actually be exposed on the module."""
    for name in codepathfinder.__all__:
        assert hasattr(codepathfinder, name), f"{name!r} listed in __all__ but missing"


def test_version_is_a_string() -> None:
    assert isinstance(codepathfinder.__version__, str)
    assert codepathfinder.__version__.count(".") >= 2
