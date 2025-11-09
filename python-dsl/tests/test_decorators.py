"""Tests for pathfinder.decorators module."""

from codepathfinder import rule, calls, variable
from codepathfinder.decorators import Rule


class TestRuleDecorator:
    """Test suite for @rule decorator."""

    def test_basic_rule(self):
        """Test basic @rule decorator usage."""

        @rule(id="test-001", severity="high")
        def detect_eval():
            """Detects eval calls"""
            return calls("eval")

        assert isinstance(detect_eval, Rule)
        assert detect_eval.id == "test-001"
        assert detect_eval.severity == "high"
        assert detect_eval.name == "detect_eval"
        assert detect_eval.description == "Detects eval calls"

    def test_rule_with_cwe_owasp(self):
        """Test @rule with CWE and OWASP metadata."""

        @rule(id="sqli", severity="critical", cwe="CWE-89", owasp="A03:2021")
        def detect_sqli():
            """SQL injection detection"""
            return calls("execute")

        assert detect_sqli.cwe == "CWE-89"
        assert detect_sqli.owasp == "A03:2021"

    def test_rule_execute(self):
        """Test Rule.execute() generates correct JSON IR."""

        @rule(id="test", severity="medium")
        def detect_test():
            """Test rule"""
            return calls("test_func")

        result = detect_test.execute()

        # Check rule metadata
        assert result["rule"]["id"] == "test"
        assert result["rule"]["name"] == "detect_test"
        assert result["rule"]["severity"] == "medium"
        assert result["rule"]["description"] == "Test rule"

        # Check matcher IR
        assert result["matcher"]["type"] == "call_matcher"
        assert result["matcher"]["patterns"] == ["test_func"]

    def test_rule_with_variable_matcher(self):
        """Test rule returning variable matcher."""

        @rule(id="var-test", severity="low")
        def detect_var():
            return variable("user_input")

        result = detect_var.execute()
        assert result["matcher"]["type"] == "variable_matcher"
        assert result["matcher"]["pattern"] == "user_input"

    def test_rule_without_docstring(self):
        """Test rule function without docstring has empty description."""

        @rule(id="no-doc", severity="low")
        def no_docstring():
            return calls("test")

        assert no_docstring.description == ""
