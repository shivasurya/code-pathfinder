"""Tests for pathfinder.matchers module."""

import pytest
from pathfinder import calls, variable
from pathfinder.matchers import CallMatcher, VariableMatcher


class TestCallMatcher:
    """Test suite for calls() matcher."""

    def test_single_pattern(self):
        """Test calls() with single pattern."""
        matcher = calls("eval")
        assert isinstance(matcher, CallMatcher)
        assert matcher.patterns == ["eval"]
        assert matcher.wildcard is False

    def test_multiple_patterns(self):
        """Test calls() with multiple patterns."""
        matcher = calls("eval", "exec", "compile")
        assert matcher.patterns == ["eval", "exec", "compile"]
        assert matcher.wildcard is False

    def test_wildcard_pattern(self):
        """Test calls() with wildcard."""
        matcher = calls("request.*", "*.json")
        assert matcher.patterns == ["request.*", "*.json"]
        assert matcher.wildcard is True

    def test_mixed_wildcard(self):
        """Test calls() with mixed wildcard and exact."""
        matcher = calls("eval", "request.*")
        assert matcher.wildcard is True

    def test_no_patterns_raises(self):
        """Test calls() with no patterns raises ValueError."""
        with pytest.raises(ValueError, match="at least one pattern"):
            calls()

    def test_empty_pattern_raises(self):
        """Test calls() with empty string raises ValueError."""
        with pytest.raises(ValueError, match="non-empty strings"):
            calls("")

    def test_none_pattern_raises(self):
        """Test calls() with None raises ValueError."""
        with pytest.raises(ValueError, match="non-empty strings"):
            calls(None)  # type: ignore

    def test_to_ir(self):
        """Test CallMatcher.to_ir() JSON serialization."""
        matcher = calls("eval", "exec")
        ir = matcher.to_ir()

        assert ir["type"] == "call_matcher"
        assert ir["patterns"] == ["eval", "exec"]
        assert ir["wildcard"] is False
        assert ir["match_mode"] == "any"

    def test_to_ir_wildcard(self):
        """Test CallMatcher.to_ir() with wildcard."""
        matcher = calls("request.*")
        ir = matcher.to_ir()

        assert ir["wildcard"] is True

    def test_repr(self):
        """Test CallMatcher.__repr__() output."""
        matcher = calls("eval", "exec")
        assert repr(matcher) == 'calls("eval", "exec")'


class TestVariableMatcher:
    """Test suite for variable() matcher."""

    def test_exact_pattern(self):
        """Test variable() with exact pattern."""
        matcher = variable("user_input")
        assert isinstance(matcher, VariableMatcher)
        assert matcher.pattern == "user_input"
        assert matcher.wildcard is False

    def test_wildcard_prefix(self):
        """Test variable() with wildcard prefix."""
        matcher = variable("user_*")
        assert matcher.pattern == "user_*"
        assert matcher.wildcard is True

    def test_wildcard_suffix(self):
        """Test variable() with wildcard suffix."""
        matcher = variable("*_id")
        assert matcher.wildcard is True

    def test_wildcard_middle(self):
        """Test variable() with wildcard in middle."""
        matcher = variable("user_*_id")
        assert matcher.wildcard is True

    def test_empty_pattern_raises(self):
        """Test variable() with empty string raises ValueError."""
        with pytest.raises(ValueError, match="non-empty string pattern"):
            variable("")

    def test_none_pattern_raises(self):
        """Test variable() with None raises ValueError."""
        with pytest.raises(ValueError, match="non-empty string pattern"):
            variable(None)  # type: ignore

    def test_to_ir(self):
        """Test VariableMatcher.to_ir() JSON serialization."""
        matcher = variable("user_input")
        ir = matcher.to_ir()

        assert ir["type"] == "variable_matcher"
        assert ir["pattern"] == "user_input"
        assert ir["wildcard"] is False

    def test_to_ir_wildcard(self):
        """Test VariableMatcher.to_ir() with wildcard."""
        matcher = variable("*_id")
        ir = matcher.to_ir()

        assert ir["wildcard"] is True

    def test_repr(self):
        """Test VariableMatcher.__repr__() output."""
        matcher = variable("user_input")
        assert repr(matcher) == 'variable("user_input")'
