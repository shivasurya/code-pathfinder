"""Tests for logic operators."""

import pytest
from codepathfinder import calls, variable, flows, And, Or, Not, propagates
from codepathfinder.logic import AndOperator, OrOperator, NotOperator
from codepathfinder.ir import IRType


class TestAndOperator:
    """Tests for And operator."""

    def test_and_two_matchers(self):
        """Test And with two matchers."""
        matcher = And(calls("eval"), calls("exec"))
        assert isinstance(matcher, AndOperator)
        assert len(matcher.matchers) == 2

    def test_and_three_matchers(self):
        """Test And with three matchers."""
        matcher = And(calls("eval"), calls("exec"), variable("user_input"))
        assert len(matcher.matchers) == 3

    def test_and_requires_two_matchers(self):
        """Test And raises ValueError with less than 2 matchers."""
        with pytest.raises(ValueError, match="requires at least 2 matchers"):
            And(calls("eval"))

    def test_and_empty_raises(self):
        """Test And raises ValueError with no matchers."""
        with pytest.raises(ValueError, match="requires at least 2 matchers"):
            And()

    def test_and_to_ir(self):
        """Test And serializes to correct IR."""
        matcher = And(calls("eval"), variable("user_input"))
        ir = matcher.to_ir()
        assert ir["type"] == IRType.LOGIC_AND.value
        assert ir["type"] == "logic_and"
        assert len(ir["matchers"]) == 2
        assert ir["matchers"][0]["type"] == "call_matcher"
        assert ir["matchers"][1]["type"] == "variable_matcher"

    def test_and_repr(self):
        """Test And __repr__."""
        matcher = And(calls("eval"), calls("exec"))
        assert repr(matcher) == "And(2 matchers)"

    def test_and_three_matchers_repr(self):
        """Test And __repr__ with three matchers."""
        matcher = And(calls("eval"), calls("exec"), variable("x"))
        assert repr(matcher) == "And(3 matchers)"


class TestOrOperator:
    """Tests for Or operator."""

    def test_or_two_matchers(self):
        """Test Or with two matchers."""
        matcher = Or(calls("eval"), calls("exec"))
        assert isinstance(matcher, OrOperator)
        assert len(matcher.matchers) == 2

    def test_or_three_matchers(self):
        """Test Or with three matchers."""
        matcher = Or(calls("eval"), calls("exec"), calls("compile"))
        assert len(matcher.matchers) == 3

    def test_or_requires_two_matchers(self):
        """Test Or raises ValueError with less than 2 matchers."""
        with pytest.raises(ValueError, match="requires at least 2 matchers"):
            Or(calls("eval"))

    def test_or_empty_raises(self):
        """Test Or raises ValueError with no matchers."""
        with pytest.raises(ValueError, match="requires at least 2 matchers"):
            Or()

    def test_or_to_ir(self):
        """Test Or serializes to correct IR."""
        matcher = Or(calls("eval"), calls("exec"))
        ir = matcher.to_ir()
        assert ir["type"] == IRType.LOGIC_OR.value
        assert ir["type"] == "logic_or"
        assert len(ir["matchers"]) == 2

    def test_or_repr(self):
        """Test Or __repr__."""
        matcher = Or(calls("eval"), calls("exec"))
        assert repr(matcher) == "Or(2 matchers)"


class TestNotOperator:
    """Tests for Not operator."""

    def test_not_matcher(self):
        """Test Not with single matcher."""
        matcher = Not(calls("test_*"))
        assert isinstance(matcher, NotOperator)
        assert matcher.matcher is not None

    def test_not_with_call_matcher(self):
        """Test Not with CallMatcher."""
        matcher = Not(calls("eval"))
        assert matcher.matcher is not None

    def test_not_with_variable_matcher(self):
        """Test Not with VariableMatcher."""
        matcher = Not(variable("user_input"))
        assert matcher.matcher is not None

    def test_not_to_ir(self):
        """Test Not serializes to correct IR."""
        matcher = Not(calls("eval"))
        ir = matcher.to_ir()
        assert ir["type"] == IRType.LOGIC_NOT.value
        assert ir["type"] == "logic_not"
        assert "matcher" in ir
        assert ir["matcher"]["type"] == "call_matcher"

    def test_not_repr(self):
        """Test Not __repr__."""
        matcher = Not(calls("test_*"))
        assert "Not(" in repr(matcher)
        assert "call_matcher" in repr(matcher).lower() or "calls" in repr(matcher)


class TestLogicNesting:
    """Tests for nested logic operators."""

    def test_and_of_ors(self):
        """Test And containing Or operators."""
        matcher = And(
            Or(calls("eval"), calls("exec")),
            Or(variable("user_input"), variable("user_data")),
        )
        assert len(matcher.matchers) == 2
        ir = matcher.to_ir()
        assert ir["type"] == "logic_and"
        assert ir["matchers"][0]["type"] == "logic_or"
        assert ir["matchers"][1]["type"] == "logic_or"

    def test_or_of_ands(self):
        """Test Or containing And operators."""
        matcher = Or(
            And(calls("eval"), variable("user_input")),
            And(calls("exec"), variable("user_data")),
        )
        assert len(matcher.matchers) == 2
        ir = matcher.to_ir()
        assert ir["type"] == "logic_or"

    def test_not_of_and(self):
        """Test Not containing And operator."""
        matcher = Not(And(calls("test_*"), calls("pytest.*")))
        ir = matcher.to_ir()
        assert ir["type"] == "logic_not"
        assert ir["matcher"]["type"] == "logic_and"

    def test_not_of_or(self):
        """Test Not containing Or operator."""
        matcher = Not(Or(calls("test_*"), calls("*_test")))
        ir = matcher.to_ir()
        assert ir["type"] == "logic_not"
        assert ir["matcher"]["type"] == "logic_or"


class TestLogicWithDataflow:
    """Tests for logic operators with dataflow matchers."""

    def test_or_of_dataflow_matchers(self):
        """Test Or combining two dataflow matchers."""
        sql_injection = flows(
            from_sources=calls("request.GET"),
            to_sinks=calls("execute"),
            propagates_through=[propagates.assignment()],
        )

        command_injection = flows(
            from_sources=calls("request.POST"),
            to_sinks=calls("os.system"),
            propagates_through=[propagates.assignment()],
        )

        matcher = Or(sql_injection, command_injection)
        assert len(matcher.matchers) == 2

        ir = matcher.to_ir()
        assert ir["type"] == "logic_or"
        assert len(ir["matchers"]) == 2
        assert ir["matchers"][0]["type"] == "dataflow"
        assert ir["matchers"][1]["type"] == "dataflow"

    def test_and_with_dataflow_and_call(self):
        """Test And combining dataflow matcher with call matcher."""
        dataflow = flows(
            from_sources=calls("request.GET"),
            to_sinks=calls("eval"),
            propagates_through=[],
        )

        matcher = And(dataflow, Not(calls("test_*")))
        assert len(matcher.matchers) == 2

        ir = matcher.to_ir()
        assert ir["type"] == "logic_and"
        assert ir["matchers"][0]["type"] == "dataflow"
        assert ir["matchers"][1]["type"] == "logic_not"


class TestLogicIntegration:
    """Integration tests for realistic logic patterns."""

    def test_sql_or_command_injection(self):
        """Test detecting SQL OR command injection."""
        matcher = Or(
            flows(
                from_sources=calls("request.*"),
                to_sinks=calls("execute", "executemany"),
                propagates_through=[propagates.assignment()],
            ),
            flows(
                from_sources=calls("request.*"),
                to_sinks=calls("os.system", "subprocess.call"),
                propagates_through=[propagates.assignment()],
            ),
        )

        ir = matcher.to_ir()
        assert ir["type"] == "logic_or"
        assert len(ir["matchers"]) == 2

    def test_eval_not_in_tests(self):
        """Test detecting eval calls NOT in test files."""
        matcher = And(
            calls("eval", "exec"),
            Not(calls("test_*", "*_test", "pytest.*")),
        )

        ir = matcher.to_ir()
        assert ir["type"] == "logic_and"
        assert len(ir["matchers"]) == 2
        assert ir["matchers"][0]["type"] == "call_matcher"
        assert ir["matchers"][1]["type"] == "logic_not"

    def test_complex_nested_logic(self):
        """Test complex nested logic expression."""
        matcher = And(
            Or(calls("eval"), calls("exec")),
            Not(Or(calls("test_*"), variable("_test_*"))),
            variable("user_*"),
        )

        ir = matcher.to_ir()
        assert ir["type"] == "logic_and"
        assert len(ir["matchers"]) == 3


class TestLogicPublicAPI:
    """Tests for public API functions."""

    def test_and_function_returns_operator(self):
        """Test And function returns AndOperator."""
        result = And(calls("eval"), calls("exec"))
        assert isinstance(result, AndOperator)

    def test_or_function_returns_operator(self):
        """Test Or function returns OrOperator."""
        result = Or(calls("eval"), calls("exec"))
        assert isinstance(result, OrOperator)

    def test_not_function_returns_operator(self):
        """Test Not function returns NotOperator."""
        result = Not(calls("eval"))
        assert isinstance(result, NotOperator)
