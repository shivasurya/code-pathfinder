"""Tests for Go rule decorator, IR compiler, and QueryType classes."""

import json

import pytest
from codepathfinder import calls, flows
from codepathfinder.go_rule import (
    GoHTTPRequest,
    GoSQLDB,
    GoOS,
    GoOSExec,
    GoFmt,
    GoIO,
    GoFilepath,
    GoStrconv,
    GoJSON,
    GoTemplate,
    GoContext,
    GoCrypto,
    GoHTTPClient,
    GoHTTPResponseWriter,
)
from codepathfinder.presets import PropagationPresets
from rules.go_decorators import go_rule, get_go_rules, clear_go_rules
from rules.go_ir import compile_go_rules


@pytest.fixture(autouse=True)
def _clear_rules():
    """Clear registered rules before each test."""
    clear_go_rules()
    yield
    clear_go_rules()


# ========== QueryType Class Tests ==========


class TestGoQueryTypes:
    def test_http_request_method(self):
        matcher = GoHTTPRequest.method("FormValue")
        ir = matcher.to_ir()
        assert ir["type"] == "type_constrained_call"
        assert "net/http.Request" in ir["receiverTypes"]
        assert "FormValue" in ir["methodNames"]

    def test_http_request_attr(self):
        matcher = GoHTTPRequest.attr("URL.Path", "Host")
        ir = matcher.to_ir()
        assert ir["type"] == "type_constrained_attribute"
        assert "net/http.Request" in ir["receiverTypes"]
        assert "URL.Path" in ir["attributeNames"]
        assert "Host" in ir["attributeNames"]

    def test_sql_db_method_with_tracks(self):
        matcher = GoSQLDB.method("Query", "Exec").tracks(0)
        ir = matcher.to_ir()
        assert ir["type"] == "type_constrained_call"
        assert "database/sql.DB" in ir["receiverTypes"]
        assert "Query" in ir["methodNames"]
        assert "Exec" in ir["methodNames"]
        assert len(ir["trackedParams"]) == 1
        assert ir["trackedParams"][0]["index"] == 0

    def test_os_exec_method(self):
        matcher = GoOSExec.method("Command")
        ir = matcher.to_ir()
        has_exec = any("os/exec" in fqn for fqn in ir["receiverTypes"])
        assert has_exec

    def test_filepath_method(self):
        matcher = GoFilepath.method("Clean", "Base")
        ir = matcher.to_ir()
        assert "path/filepath" in ir["receiverTypes"]
        assert "Clean" in ir["methodNames"]
        assert "Base" in ir["methodNames"]

    def test_all_query_types_have_fqns(self):
        """Every QueryType class must have non-empty fqns."""
        types = [
            GoHTTPRequest,
            GoHTTPClient,
            GoHTTPResponseWriter,
            GoSQLDB,
            GoOS,
            GoOSExec,
            GoFmt,
            GoIO,
            GoFilepath,
            GoStrconv,
            GoJSON,
            GoTemplate,
            GoContext,
            GoCrypto,
        ]
        for qt in types:
            assert len(qt.fqns) > 0, f"{qt.__name__} must have non-empty fqns"
            assert all(
                isinstance(f, str) for f in qt.fqns
            ), f"{qt.__name__} fqns must be strings"


# ========== @go_rule Decorator Tests ==========


class TestGoRuleDecorator:
    def test_basic_rule(self):
        @go_rule(id="TEST-001", severity="HIGH", cwe="CWE-89")
        def test_sqli():
            return flows(
                from_sources=[calls("FormValue")],
                to_sinks=[calls("*Query")],
                propagates_through=PropagationPresets.standard(),
                scope="global",
            )

        rules = get_go_rules()
        assert len(rules) == 1
        assert rules[0].metadata.id == "TEST-001"
        assert rules[0].metadata.severity == "HIGH"
        assert rules[0].metadata.cwe == "CWE-89"

    def test_language_injected(self):
        @go_rule(id="TEST-002", severity="MEDIUM")
        def test_rule():
            return flows(
                from_sources=[calls("FormValue")],
                to_sinks=[calls("*Query")],
                propagates_through=PropagationPresets.standard(),
                scope="local",
            )

        rules = get_go_rules()
        matcher = rules[0].matcher
        assert matcher["language"] == "go", "Matcher dict must have language='go'"
        assert matcher["type"] == "dataflow"

    def test_full_metadata(self):
        @go_rule(
            id="GO-NET-001",
            name="Go SSRF",
            severity="HIGH",
            category="net-http",
            cwe="CWE-918",
            cve="CVE-2024-1234",
            tags="go,ssrf",
            message="User input flows to http.Get",
            owasp="A10:2021",
        )
        def test_ssrf():
            return flows(
                from_sources=[GoHTTPRequest.method("FormValue")],
                to_sinks=[GoHTTPClient.method("Get")],
                propagates_through=PropagationPresets.standard(),
                scope="global",
            )

        rules = get_go_rules()
        meta = rules[0].metadata
        assert meta.name == "Go SSRF"
        assert meta.category == "net-http"
        assert meta.cve == "CVE-2024-1234"
        assert meta.tags == "go,ssrf"
        assert meta.owasp == "A10:2021"

    def test_non_dataflow_matcher(self):
        """Non-dataflow matchers should not get language field."""

        @go_rule(id="TEST-003", severity="LOW")
        def test_call_only():
            return calls("*Query")

        rules = get_go_rules()
        matcher = rules[0].matcher
        assert matcher["type"] == "call_matcher"
        assert "language" not in matcher


# ========== go_ir.py Compiler Tests ==========


class TestGoIRCompiler:
    def test_compile_empty(self):
        result = compile_go_rules()
        assert result == []

    def test_compile_single_rule(self):
        @go_rule(id="GO-001", severity="CRITICAL", cwe="CWE-89")
        def test_sqli():
            return flows(
                from_sources=[GoHTTPRequest.method("FormValue")],
                to_sinks=[GoSQLDB.method("Query").tracks(0)],
                propagates_through=PropagationPresets.standard(),
                scope="global",
            )

        result = compile_go_rules()
        assert len(result) == 1

        rule_ir = result[0]
        assert rule_ir["rule"]["id"] == "GO-001"
        assert rule_ir["rule"]["severity"] == "critical"
        assert rule_ir["rule"]["cwe"] == "CWE-89"
        assert rule_ir["rule"]["language"] == "go"

        matcher = rule_ir["matcher"]
        assert matcher["type"] == "dataflow"
        assert matcher["language"] == "go"
        assert matcher["scope"] == "global"
        assert len(matcher["sources"]) == 1
        assert len(matcher["sinks"]) == 1
        assert matcher["sources"][0]["type"] == "type_constrained_call"
        assert matcher["sinks"][0]["type"] == "type_constrained_call"

    def test_compile_json_serializable(self):
        @go_rule(id="GO-002", severity="HIGH")
        def test_rule():
            return flows(
                from_sources=[GoHTTPRequest.attr("URL.Path")],
                to_sinks=[GoOS.method("WriteFile")],
                propagates_through=PropagationPresets.standard(),
                scope="global",
            )

        result = compile_go_rules()
        json_str = json.dumps(result)
        parsed = json.loads(json_str)
        assert len(parsed) == 1

        source = parsed[0]["matcher"]["sources"][0]
        assert source["type"] == "type_constrained_attribute"
        assert "net/http.Request" in source["receiverTypes"]
        assert "URL.Path" in source["attributeNames"]
