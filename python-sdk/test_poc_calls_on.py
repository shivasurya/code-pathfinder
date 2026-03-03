#!/usr/bin/env python3
"""PoC validation: calls_on() generates correct JSON IR for Go executor."""

import json
import sys

# Add parent to path for local development
sys.path.insert(0, ".")

from codepathfinder import calls_on, rule


def test_basic_ir():
    """Test that calls_on generates correct IR."""
    matcher = calls_on("Cursor", "execute")
    ir = matcher.to_ir()

    assert ir["type"] == "type_constrained_call", f"Expected type_constrained_call, got {ir['type']}"
    assert ir["receiverType"] == "Cursor", f"Expected Cursor, got {ir['receiverType']}"
    assert ir["methodName"] == "execute", f"Expected execute, got {ir['methodName']}"
    assert ir["minConfidence"] == 0.5, f"Expected 0.5, got {ir['minConfidence']}"
    assert ir["fallbackMode"] == "name", f"Expected name, got {ir['fallbackMode']}"

    print("PASS: basic IR generation")
    print(f"  IR: {json.dumps(ir, indent=2)}")


def test_custom_params():
    """Test calls_on with custom confidence and fallback."""
    matcher = calls_on("sqlite3.Cursor", "execute", min_confidence=0.8, fallback="none")
    ir = matcher.to_ir()

    assert ir["receiverType"] == "sqlite3.Cursor"
    assert ir["minConfidence"] == 0.8
    assert ir["fallbackMode"] == "none"

    print("PASS: custom parameters")


def test_repr():
    """Test string representation."""
    matcher = calls_on("Cursor", "execute")
    assert repr(matcher) == 'calls_on("Cursor", "execute")'
    print("PASS: repr")


def test_rule_integration():
    """Test calls_on works with @rule decorator."""

    @rule(id="poc-sql-injection", severity="CRITICAL", cwe="CWE-89")
    def detect_sql_injection():
        """Detect SQL injection via type-aware cursor matching."""
        return calls_on("Cursor", "execute", fallback="none")

    # Execute the rule to get full IR
    result = detect_sql_injection.execute()

    assert result["rule"]["id"] == "poc-sql-injection"
    assert result["matcher"]["type"] == "type_constrained_call"
    assert result["matcher"]["receiverType"] == "Cursor"
    assert result["matcher"]["methodName"] == "execute"

    print("PASS: @rule integration")
    print(f"  Full rule IR: {json.dumps(result, indent=2)}")


if __name__ == "__main__":
    test_basic_ir()
    test_custom_params()
    test_repr()
    test_rule_integration()
    print("\nAll PoC tests passed!")
