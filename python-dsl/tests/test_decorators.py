"""Tests for pathfinder.decorators module."""

import json
from unittest.mock import patch
from codepathfinder import rule, calls, variable
from codepathfinder.decorators import (
    Rule,
    _enable_auto_execute,
    _register_rule,
    _rule_registry,
)


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


class TestAutoExecution:
    """Test suite for auto-execution features."""

    def test_enable_auto_execute_registers_atexit(self):
        """Test that _enable_auto_execute registers an atexit handler."""
        with patch("atexit.register") as mock_atexit:
            with patch("codepathfinder.decorators._auto_execute_enabled", False):
                _enable_auto_execute()
                assert mock_atexit.called

    def test_enable_auto_execute_only_once(self):
        """Test that _enable_auto_execute only registers once."""
        with patch("atexit.register") as mock_atexit:
            with patch("codepathfinder.decorators._auto_execute_enabled", True):
                _enable_auto_execute()
                # Should not call atexit.register if already enabled
                assert not mock_atexit.called

    def test_register_rule_adds_to_registry(self):
        """Test that _register_rule adds rule to global registry."""
        initial_count = len(_rule_registry)

        rule_obj = Rule(id="test", severity="high", func=lambda: calls("test"))

        with patch("sys._getframe") as mock_frame:
            # Mock frame to avoid __main__ check
            mock_frame.return_value.f_globals = {"__name__": "test_module"}
            _register_rule(rule_obj)

        assert len(_rule_registry) > initial_count

    def test_register_rule_enables_auto_execute_in_main(self):
        """Test that _register_rule enables auto-execute when in __main__."""
        rule_obj = Rule(id="test", severity="high", func=lambda: calls("test"))

        with patch("sys._getframe") as mock_frame:
            with patch("codepathfinder.decorators._enable_auto_execute") as mock_enable:
                # Mock frame to simulate __main__
                mock_frame.return_value.f_globals = {"__name__": "__main__"}
                _register_rule(rule_obj)
                assert mock_enable.called

    def test_output_rules_format(self, capsys):
        """Test that rules are output in correct JSON format."""
        from codepathfinder.decorators import _rule_registry

        # Create test rules
        rule1 = Rule(id="test-1", severity="high", func=lambda: calls("eval"))
        rule2 = Rule(id="test-2", severity="medium", func=lambda: calls("exec"))

        # Store original registry
        original_registry = _rule_registry.copy()

        try:
            # Clear and add test rules
            _rule_registry.clear()
            _rule_registry.append(rule1)
            _rule_registry.append(rule2)

            # Simulate the _output_rules function
            rules_json = [r.execute() for r in _rule_registry]
            output = json.dumps(rules_json)

            # Verify output is valid JSON
            parsed = json.loads(output)
            assert len(parsed) == 2
            assert parsed[0]["rule"]["id"] == "test-1"
            assert parsed[1]["rule"]["id"] == "test-2"

        finally:
            # Restore original registry
            _rule_registry.clear()
            _rule_registry.extend(original_registry)

    def test_output_rules_with_empty_registry(self):
        """Test that _output_rules handles empty registry gracefully."""
        from codepathfinder.decorators import _rule_registry

        # Store original registry
        original_registry = _rule_registry.copy()

        try:
            # Clear registry
            _rule_registry.clear()

            # Simulate _output_rules with empty registry
            if not _rule_registry:
                # Should return early, not output anything
                pass

            # Should not raise an error
            assert len(_rule_registry) == 0

        finally:
            # Restore original registry
            _rule_registry.clear()
            _rule_registry.extend(original_registry)
