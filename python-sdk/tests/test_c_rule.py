"""Tests for the @c_rule decorator and the C IR compiler."""

import json

import pytest

from codepathfinder import calls, flows
from codepathfinder.presets import PropagationPresets
from codepathfinder.c_decorators import c_rule, get_c_rules, clear_c_rules
from codepathfinder.c_ir import compile_c_rules, compile_all_rules


@pytest.fixture(autouse=True)
def _clear_rules():
    """Reset the global rule registry around every test."""
    clear_c_rules()
    yield
    clear_c_rules()


# ========== Decorator metadata + registration ==========


class TestCRuleDecorator:
    def test_basic_rule_registers_once(self):
        @c_rule(id="C-TEST-001", severity="HIGH", cwe="CWE-78")
        def c_command_injection():
            return calls("system", "popen")

        rules = get_c_rules()
        assert len(rules) == 1
        assert rules[0].metadata.id == "C-TEST-001"
        assert rules[0].metadata.severity == "HIGH"
        assert rules[0].metadata.cwe == "CWE-78"

    def test_default_name_derived_from_func(self):
        @c_rule(id="C-TEST-002")
        def c_unsafe_string_copy():
            return calls("strcpy")

        rules = get_c_rules()
        assert rules[0].metadata.name == "C Unsafe String Copy"

    def test_explicit_name_wins(self):
        @c_rule(id="C-TEST-003", name="Override Name")
        def c_anything():
            return calls("foo")

        assert get_c_rules()[0].metadata.name == "Override Name"

    def test_full_metadata(self):
        @c_rule(
            id="C-NET-001",
            name="C SSRF",
            severity="HIGH",
            category="net",
            cwe="CWE-918",
            cve="CVE-2024-9999",
            tags="c,ssrf",
            message="User input flows to network call",
            owasp="A10:2021",
        )
        def c_ssrf():
            return flows(
                from_sources=[calls("recv")],
                to_sinks=[calls("connect")],
                propagates_through=PropagationPresets.standard(),
                scope="global",
            )

        meta = get_c_rules()[0].metadata
        assert meta.name == "C SSRF"
        assert meta.category == "net"
        assert meta.cve == "CVE-2024-9999"
        assert meta.tags == "c,ssrf"
        assert meta.owasp == "A10:2021"
        assert meta.message == "User input flows to network call"

    def test_default_message_when_missing(self):
        @c_rule(id="C-MSG-001")
        def c_default_msg():
            return calls("strcpy")

        assert (
            get_c_rules()[0].metadata.message
            == "Security issue detected by C-MSG-001"
        )

    def test_returns_underlying_function(self):
        @c_rule(id="C-RET-001")
        def c_identity():
            return calls("strcpy")

        # Decorator must preserve the original callable so atexit + repeated
        # invocations work the same as @go_rule.
        assert callable(c_identity)
        assert c_identity.__name__ == "c_identity"


# ========== Language injection contract ==========


class TestLanguageInjection:
    def test_language_injected_into_dataflow(self):
        @c_rule(id="C-DF-001", severity="MEDIUM")
        def c_buffer_overflow():
            return flows(
                from_sources=[calls("gets", "scanf")],
                to_sinks=[calls("strcpy", "strcat")],
                propagates_through=PropagationPresets.standard(),
                scope="local",
            )

        matcher = get_c_rules()[0].matcher
        assert matcher["type"] == "dataflow"
        assert matcher["language"] == "c"

    def test_language_NOT_injected_for_call_matcher(self):
        """Pure calls() rules are language-agnostic — same as @go_rule."""

        @c_rule(id="C-CM-001", severity="LOW")
        def c_calls_only():
            return calls("system")

        matcher = get_c_rules()[0].matcher
        assert matcher["type"] == "call_matcher"
        assert "language" not in matcher

    def test_dict_matcher_is_passed_through(self):
        @c_rule(id="C-DICT-001")
        def c_raw_dict():
            return {"type": "dataflow", "sources": [], "sinks": []}

        matcher = get_c_rules()[0].matcher
        assert matcher["language"] == "c"

    def test_invalid_matcher_raises(self):
        with pytest.raises(ValueError, match="C-BAD-001"):

            @c_rule(id="C-BAD-001")
            def c_bad():
                return 42  # not a matcher / dict


# ========== c_ir.compile_c_rules ==========


class TestCIRCompiler:
    def test_compile_empty(self):
        assert compile_c_rules() == []
        assert compile_all_rules() == []

    def test_compile_single_dataflow_rule(self):
        @c_rule(id="C-001", severity="CRITICAL", cwe="CWE-120", owasp="A03:2021")
        def c_buffer_overflow():
            return flows(
                from_sources=[calls("gets")],
                to_sinks=[calls("strcpy")],
                propagates_through=PropagationPresets.standard(),
                scope="global",
            )

        result = compile_c_rules()
        assert len(result) == 1

        ir = result[0]
        assert ir["rule"]["id"] == "C-001"
        assert ir["rule"]["severity"] == "critical"
        assert ir["rule"]["cwe"] == "CWE-120"
        assert ir["rule"]["owasp"] == "A03:2021"
        assert ir["rule"]["language"] == "c"

        matcher = ir["matcher"]
        assert matcher["type"] == "dataflow"
        assert matcher["language"] == "c"
        assert matcher["scope"] == "global"

    def test_compile_call_matcher_rule_keeps_metadata_language(self):
        """`rule.language` is "c" even when the matcher is a pure calls() one."""

        @c_rule(id="C-002", severity="HIGH")
        def c_format_string():
            return calls("printf", "sprintf")

        ir = compile_c_rules()[0]
        assert ir["rule"]["language"] == "c"
        assert ir["matcher"]["type"] == "call_matcher"
        assert "language" not in ir["matcher"]

    def test_compile_default_description_when_message_missing(self):
        # The decorator fills metadata.message with a default when blank,
        # so compile_c_rules must surface that as the IR description.
        @c_rule(id="C-NOMSG-001")
        def c_nomsg():
            return calls("strcpy")

        ir = compile_c_rules()[0]
        assert ir["rule"]["description"] == "Security issue detected by C-NOMSG-001"

    def test_compile_json_serializable(self):
        @c_rule(id="C-JSON-001", severity="HIGH")
        def c_json_round_trip():
            return flows(
                from_sources=[calls("recv")],
                to_sinks=[calls("strcpy")],
                propagates_through=PropagationPresets.standard(),
                scope="global",
            )

        encoded = json.dumps(compile_c_rules())
        parsed = json.loads(encoded)
        assert parsed[0]["rule"]["language"] == "c"
        assert parsed[0]["matcher"]["language"] == "c"


# ========== Registry hygiene ==========


class TestRegistryIsolation:
    def test_clear_resets_state(self):
        @c_rule(id="C-X-001")
        def c_x():
            return calls("strcpy")

        assert len(get_c_rules()) == 1
        clear_c_rules()
        assert get_c_rules() == []

    def test_get_returns_a_copy(self):
        @c_rule(id="C-COPY-001")
        def c_copy():
            return calls("strcpy")

        snapshot = get_c_rules()
        snapshot.clear()
        assert len(get_c_rules()) == 1, "external mutation must not affect registry"
