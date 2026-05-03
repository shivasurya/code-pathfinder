"""Tests for the @cpp_rule decorator and the C++ IR compiler."""

import json

import pytest

from codepathfinder import calls, flows
from codepathfinder.presets import PropagationPresets
from codepathfinder.cpp_decorators import cpp_rule, get_cpp_rules, clear_cpp_rules
from codepathfinder.cpp_ir import compile_cpp_rules, compile_all_rules


@pytest.fixture(autouse=True)
def _clear_rules():
    """Reset the global rule registry around every test."""
    clear_cpp_rules()
    yield
    clear_cpp_rules()


# ========== Decorator metadata + registration ==========


class TestCppRuleDecorator:
    def test_basic_rule_registers_once(self):
        @cpp_rule(id="CPP-TEST-001", severity="HIGH", cwe="CWE-78")
        def cpp_command_injection():
            return calls("system", "popen")

        rules = get_cpp_rules()
        assert len(rules) == 1
        assert rules[0].metadata.id == "CPP-TEST-001"
        assert rules[0].metadata.severity == "HIGH"

    def test_default_name_derived_from_func(self):
        @cpp_rule(id="CPP-TEST-002")
        def cpp_unsafe_resource():
            return calls("fopen")

        assert get_cpp_rules()[0].metadata.name == "Cpp Unsafe Resource"

    def test_full_metadata(self):
        @cpp_rule(
            id="CPP-NET-001",
            name="C++ SSRF",
            severity="HIGH",
            category="net",
            cwe="CWE-918",
            cve="CVE-2024-9999",
            tags="cpp,ssrf",
            message="User input flows to network call",
            owasp="A10:2021",
        )
        def cpp_ssrf():
            return flows(
                from_sources=[calls("recv")],
                to_sinks=[calls("connect")],
                propagates_through=PropagationPresets.standard(),
                scope="global",
            )

        meta = get_cpp_rules()[0].metadata
        assert meta.name == "C++ SSRF"
        assert meta.cve == "CVE-2024-9999"
        assert meta.tags == "cpp,ssrf"
        assert meta.owasp == "A10:2021"

    def test_default_message_when_missing(self):
        @cpp_rule(id="CPP-MSG-001")
        def cpp_default_msg():
            return calls("strcpy")

        assert (
            get_cpp_rules()[0].metadata.message
            == "Security issue detected by CPP-MSG-001"
        )

    def test_returns_underlying_function(self):
        @cpp_rule(id="CPP-RET-001")
        def cpp_identity():
            return calls("strcpy")

        assert callable(cpp_identity)
        assert cpp_identity.__name__ == "cpp_identity"


# ========== Language injection contract ==========


class TestLanguageInjection:
    def test_language_injected_into_dataflow(self):
        @cpp_rule(id="CPP-DF-001")
        def cpp_resource_management():
            return flows(
                from_sources=[calls("fopen")],
                to_sinks=[calls("write")],
                propagates_through=PropagationPresets.standard(),
                scope="local",
            )

        matcher = get_cpp_rules()[0].matcher
        assert matcher["type"] == "dataflow"
        assert matcher["language"] == "cpp"
        # Critical: must NOT collide with the C decorator's tag.
        assert matcher["language"] != "c"

    def test_language_NOT_injected_for_call_matcher(self):
        @cpp_rule(id="CPP-CM-001", severity="LOW")
        def cpp_calls_only():
            return calls("system")

        matcher = get_cpp_rules()[0].matcher
        assert matcher["type"] == "call_matcher"
        assert "language" not in matcher

    def test_dict_matcher_is_passed_through(self):
        @cpp_rule(id="CPP-DICT-001")
        def cpp_raw_dict():
            return {"type": "dataflow", "sources": [], "sinks": []}

        matcher = get_cpp_rules()[0].matcher
        assert matcher["language"] == "cpp"

    def test_invalid_matcher_raises(self):
        with pytest.raises(ValueError, match="CPP-BAD-001"):

            @cpp_rule(id="CPP-BAD-001")
            def cpp_bad():
                return 42  # not a matcher / dict


# ========== cpp_ir.compile_cpp_rules ==========


class TestCppIRCompiler:
    def test_compile_empty(self):
        assert compile_cpp_rules() == []
        assert compile_all_rules() == []

    def test_compile_single_dataflow_rule(self):
        @cpp_rule(id="CPP-001", severity="CRITICAL", cwe="CWE-120", owasp="A03:2021")
        def cpp_buffer_overflow():
            return flows(
                from_sources=[calls("gets")],
                to_sinks=[calls("strcpy")],
                propagates_through=PropagationPresets.standard(),
                scope="global",
            )

        ir = compile_cpp_rules()[0]
        assert ir["rule"]["id"] == "CPP-001"
        assert ir["rule"]["severity"] == "critical"
        assert ir["rule"]["language"] == "cpp"

        matcher = ir["matcher"]
        assert matcher["type"] == "dataflow"
        assert matcher["language"] == "cpp"

    def test_compile_default_description_when_message_missing(self):
        # Decorator already fills metadata.message with a default when blank;
        # compile_cpp_rules must surface it as the IR description.
        @cpp_rule(id="CPP-NOMSG-001")
        def cpp_nomsg():
            return calls("strcpy")

        ir = compile_cpp_rules()[0]
        assert (
            ir["rule"]["description"] == "Security issue detected by CPP-NOMSG-001"
        )

    def test_compile_json_serializable(self):
        @cpp_rule(id="CPP-JSON-001", severity="HIGH")
        def cpp_json_round_trip():
            return flows(
                from_sources=[calls("recv")],
                to_sinks=[calls("strcpy")],
                propagates_through=PropagationPresets.standard(),
                scope="global",
            )

        encoded = json.dumps(compile_cpp_rules())
        parsed = json.loads(encoded)
        assert parsed[0]["rule"]["language"] == "cpp"
        assert parsed[0]["matcher"]["language"] == "cpp"


# ========== Registry hygiene + decorator independence ==========


class TestRegistryIsolation:
    def test_clear_resets_state(self):
        @cpp_rule(id="CPP-X-001")
        def cpp_x():
            return calls("strcpy")

        assert len(get_cpp_rules()) == 1
        clear_cpp_rules()
        assert get_cpp_rules() == []

    def test_get_returns_a_copy(self):
        @cpp_rule(id="CPP-COPY-001")
        def cpp_copy():
            return calls("strcpy")

        snapshot = get_cpp_rules()
        snapshot.clear()
        assert len(get_cpp_rules()) == 1, "external mutation must not affect registry"

    def test_c_and_cpp_registries_are_independent(self):
        from codepathfinder.c_decorators import (
            c_rule,
            get_c_rules,
            clear_c_rules,
        )

        clear_c_rules()

        @c_rule(id="C-INDEP-001")
        def c_only():
            return calls("strcpy")

        @cpp_rule(id="CPP-INDEP-001")
        def cpp_only():
            return calls("strcpy")

        c_rules = get_c_rules()
        cpp_rules = get_cpp_rules()
        assert len(c_rules) == 1 and c_rules[0].metadata.id == "C-INDEP-001"
        assert len(cpp_rules) == 1 and cpp_rules[0].metadata.id == "CPP-INDEP-001"

        clear_cpp_rules()
        assert len(get_c_rules()) == 1, "clear_cpp_rules must not touch C registry"
        clear_c_rules()
