"""Tests for pathfinder.matchers module."""

import pytest
from codepathfinder import calls, variable
from codepathfinder.matchers import CallMatcher, VariableMatcher


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
        assert ir["matchMode"] == "any"

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


class TestCallMatcherKeywordArguments:
    """Test suite for keyword argument matching (match_name)."""

    def test_single_keyword_arg_string(self):
        """Test matching single keyword argument with string value."""
        matcher = calls("app.run", match_name={"debug": "True"})
        ir = matcher.to_ir()

        assert "keywordArgs" in ir
        assert "debug" in ir["keywordArgs"]
        assert ir["keywordArgs"]["debug"]["value"] == "True"
        assert ir["keywordArgs"]["debug"]["wildcard"] is False

    def test_single_keyword_arg_boolean(self):
        """Test matching keyword argument with boolean value."""
        matcher = calls("app.run", match_name={"debug": True})
        ir = matcher.to_ir()

        assert ir["keywordArgs"]["debug"]["value"] is True

    def test_single_keyword_arg_number(self):
        """Test matching keyword argument with numeric value."""
        matcher = calls("app.listen", match_name={"port": 8080})
        ir = matcher.to_ir()

        assert ir["keywordArgs"]["port"]["value"] == 8080

    def test_multiple_keyword_args(self):
        """Test matching multiple keyword arguments."""
        matcher = calls(
            "app.run", match_name={"host": "0.0.0.0", "port": 5000, "debug": True}
        )
        ir = matcher.to_ir()

        assert len(ir["keywordArgs"]) == 3
        assert ir["keywordArgs"]["host"]["value"] == "0.0.0.0"
        assert ir["keywordArgs"]["port"]["value"] == 5000
        assert ir["keywordArgs"]["debug"]["value"] is True

    def test_keyword_arg_or_logic(self):
        """Test matching keyword argument with multiple values (OR logic)."""
        matcher = calls(
            "yaml.load", match_name={"Loader": ["Loader", "UnsafeLoader", "FullLoader"]}
        )
        ir = matcher.to_ir()

        assert isinstance(ir["keywordArgs"]["Loader"]["value"], list)
        assert len(ir["keywordArgs"]["Loader"]["value"]) == 3


class TestCallMatcherPositionalArguments:
    """Test suite for positional argument matching (match_position)."""

    def test_single_positional_arg(self):
        """Test matching single positional argument."""
        matcher = calls("socket.bind", match_position={0: "0.0.0.0"})
        ir = matcher.to_ir()

        assert "positionalArgs" in ir
        assert "0" in ir["positionalArgs"]  # JSON keys are strings
        assert ir["positionalArgs"]["0"]["value"] == "0.0.0.0"

    def test_multiple_positional_args(self):
        """Test matching multiple positional arguments."""
        matcher = calls("chmod", match_position={0: "/tmp/file", 1: 0o777})
        ir = matcher.to_ir()

        assert len(ir["positionalArgs"]) == 2
        assert ir["positionalArgs"]["0"]["value"] == "/tmp/file"
        assert ir["positionalArgs"]["1"]["value"] == 0o777

    def test_positional_arg_or_logic(self):
        """Test matching positional argument with multiple values."""
        matcher = calls("open", match_position={1: ["w", "a", "w+", "a+"]})
        ir = matcher.to_ir()

        assert isinstance(ir["positionalArgs"]["1"]["value"], list)
        assert len(ir["positionalArgs"]["1"]["value"]) == 4


class TestCallMatcherCombinedArguments:
    """Test suite for combined positional and keyword argument matching."""

    def test_both_positional_and_keyword(self):
        """Test matching both positional and keyword arguments."""
        matcher = calls(
            "app.run",
            match_position={0: "localhost"},
            match_name={"debug": True, "port": 5000},
        )
        ir = matcher.to_ir()

        assert "positionalArgs" in ir
        assert "keywordArgs" in ir
        assert ir["positionalArgs"]["0"]["value"] == "localhost"
        assert ir["keywordArgs"]["debug"]["value"] is True
        assert ir["keywordArgs"]["port"]["value"] == 5000


class TestCallMatcherWildcardMatching:
    """Test suite for wildcard matching in argument values."""

    def test_wildcard_in_string_value(self):
        """Test automatic wildcard detection in string values."""
        matcher = calls("chmod", match_position={1: "0o7*"})
        ir = matcher.to_ir()

        # Wildcard should be auto-detected from '*' in value
        assert ir["positionalArgs"]["1"]["wildcard"] is True

    def test_wildcard_in_list_value(self):
        """Test wildcard detection in list of values."""
        matcher = calls("open", match_position={1: ["w*", "a*"]})
        ir = matcher.to_ir()

        assert ir["positionalArgs"]["1"]["wildcard"] is True

    def test_explicit_wildcard_flag(self):
        """Test explicit wildcard flag propagation."""
        matcher = calls("app.*", match_name={"host": "192.168.1.1"})
        ir = matcher.to_ir()

        assert ir["wildcard"] is True
        # Wildcard in function pattern propagates to argument constraints
        assert ir["keywordArgs"]["host"]["wildcard"] is True


class TestCallMatcherBackwardCompatibility:
    """Test suite for backward compatibility with existing rules."""

    def test_no_arguments_specified(self):
        """Test that rules without argument constraints still work."""
        matcher = calls("eval")
        ir = matcher.to_ir()

        # Should not have argument constraint fields
        assert "positionalArgs" not in ir
        assert "keywordArgs" not in ir

    def test_empty_argument_dicts(self):
        """Test that empty argument dicts don't add IR fields."""
        matcher = calls("eval", match_position={}, match_name={})
        ir = matcher.to_ir()

        # Empty dicts should not add fields
        assert "positionalArgs" not in ir
        assert "keywordArgs" not in ir


class TestCallMatcherIRSerialization:
    """Test suite for JSON serialization of generated IR."""

    def test_complex_ir_serialization(self):
        """Test that complex IR can be serialized to JSON."""
        import json

        matcher = calls(
            "app.run",
            match_position={0: "0.0.0.0"},
            match_name={"debug": True, "port": 5000, "host": ["localhost", "0.0.0.0"]},
        )
        ir = matcher.to_ir()

        # Should be JSON-serializable
        json_str = json.dumps(ir)
        reconstructed = json.loads(json_str)

        assert reconstructed["type"] == "call_matcher"
        assert reconstructed["keywordArgs"]["debug"]["value"] is True

    def test_special_values_serialization(self):
        """Test serialization of special Python values."""
        import json

        matcher = calls("chmod", match_position={1: 0o777})
        ir = matcher.to_ir()

        json_str = json.dumps(ir)
        reconstructed = json.loads(json_str)

        # Octal should be serialized as decimal integer
        assert reconstructed["positionalArgs"]["1"]["value"] == 511


class TestCallMatcherEdgeCases:
    """Test suite for edge cases and error conditions."""

    def test_none_values_handled(self):
        """Test that None match_name/match_position are handled."""
        matcher = calls("eval", match_name=None, match_position=None)
        ir = matcher.to_ir()

        assert "keywordArgs" not in ir
        assert "positionalArgs" not in ir

    def test_mixed_value_types(self):
        """Test mixing different value types in same rule."""
        matcher = calls(
            "config.set",
            match_name={
                "timeout": 30,  # int
                "enabled": True,  # bool
                "host": "localhost",  # string
                "retry": 5.5,  # float
            },
        )
        ir = matcher.to_ir()

        assert ir["keywordArgs"]["timeout"]["value"] == 30
        assert ir["keywordArgs"]["enabled"]["value"] is True
        assert ir["keywordArgs"]["host"]["value"] == "localhost"
        assert ir["keywordArgs"]["retry"]["value"] == 5.5
