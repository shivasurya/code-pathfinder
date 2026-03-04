"""Tests for sinks module — semantic sink categories."""

import json

import pytest
from codepathfinder import sinks, sources, flows, calls
from codepathfinder.sinks import (
    sql_execution,
    command_execution,
    code_execution,
    template_render,
    xpath_query,
    ldap_query,
    file_write,
    file_open,
    path_operation,
    http_request,
    socket_connect,
    deserialize,
)


class TestSqlExecution:
    """Tests for sql_execution() sink."""

    def test_returns_or_matcher(self):
        ir = sql_execution().to_ir()
        assert ir["type"] == "logic_or"

    def test_contains_cursor_execute(self):
        ir = sql_execution().to_ir()
        methods = [m.get("methodName", "") for m in ir["matchers"]]
        assert "execute" in methods

    def test_contains_executemany(self):
        ir = sql_execution().to_ir()
        methods = [m.get("methodName", "") for m in ir["matchers"]]
        assert "executemany" in methods

    def test_contains_sqlalchemy(self):
        ir = sql_execution().to_ir()
        types = [m.get("receiverType", "") for m in ir["matchers"]]
        assert "Engine" in types
        assert "Session" in types

    def test_contains_django_raw(self):
        ir = sql_execution().to_ir()
        types = [m.get("receiverType", "") for m in ir["matchers"]]
        assert "QuerySet" in types

    def test_uses_strict_fallback(self):
        """SQL sinks MUST use fallback='none' for type-constrained matchers."""
        ir = sql_execution().to_ir()
        for m in ir["matchers"]:
            if m["type"] == "type_constrained_call":
                assert m["fallbackMode"] == "none", (
                    f"SQL sink {m['receiverType']}.{m['methodName']} should use fallback='none'"
                )

    def test_matcher_count(self):
        ir = sql_execution().to_ir()
        assert len(ir["matchers"]) == 8


class TestCommandExecution:
    """Tests for command_execution() sink."""

    def test_returns_or_matcher(self):
        ir = command_execution().to_ir()
        assert ir["type"] == "logic_or"

    def test_contains_os_system(self):
        ir = command_execution().to_ir()
        call_patterns = []
        for m in ir["matchers"]:
            if m["type"] == "call_matcher":
                call_patterns.extend(m.get("patterns", []))
        assert "os.system" in call_patterns

    def test_contains_subprocess(self):
        ir = command_execution().to_ir()
        call_patterns = []
        for m in ir["matchers"]:
            if m["type"] == "call_matcher":
                call_patterns.extend(m.get("patterns", []))
        assert "subprocess.run" in call_patterns
        assert "subprocess.Popen" in call_patterns

    def test_matcher_count(self):
        ir = command_execution().to_ir()
        assert len(ir["matchers"]) == 7


class TestCodeExecution:
    """Tests for code_execution() sink."""

    def test_returns_or_matcher(self):
        ir = code_execution().to_ir()
        assert ir["type"] == "logic_or"

    def test_contains_eval_exec(self):
        ir = code_execution().to_ir()
        call_patterns = []
        for m in ir["matchers"]:
            if m["type"] == "call_matcher":
                call_patterns.extend(m.get("patterns", []))
        assert "eval" in call_patterns
        assert "exec" in call_patterns

    def test_contains_compile(self):
        ir = code_execution().to_ir()
        call_patterns = []
        for m in ir["matchers"]:
            if m["type"] == "call_matcher":
                call_patterns.extend(m.get("patterns", []))
        assert "compile" in call_patterns

    def test_matcher_count(self):
        ir = code_execution().to_ir()
        assert len(ir["matchers"]) == 4


class TestTemplateRender:
    """Tests for template_render() sink."""

    def test_returns_or_matcher(self):
        ir = template_render().to_ir()
        assert ir["type"] == "logic_or"

    def test_contains_template_render(self):
        ir = template_render().to_ir()
        methods = [m.get("methodName", "") for m in ir["matchers"]]
        assert "render" in methods

    def test_contains_mark_safe(self):
        ir = template_render().to_ir()
        call_patterns = []
        for m in ir["matchers"]:
            if m["type"] == "call_matcher":
                call_patterns.extend(m.get("patterns", []))
        assert "mark_safe" in call_patterns

    def test_uses_strict_fallback(self):
        ir = template_render().to_ir()
        for m in ir["matchers"]:
            if m["type"] == "type_constrained_call":
                assert m["fallbackMode"] == "none"

    def test_matcher_count(self):
        ir = template_render().to_ir()
        assert len(ir["matchers"]) == 4


class TestXpathQuery:
    """Tests for xpath_query() sink."""

    def test_returns_or_matcher(self):
        ir = xpath_query().to_ir()
        assert ir["type"] == "logic_or"

    def test_contains_lxml(self):
        ir = xpath_query().to_ir()
        call_patterns = []
        for m in ir["matchers"]:
            if m["type"] == "call_matcher":
                call_patterns.extend(m.get("patterns", []))
        assert "lxml.etree.parse" in call_patterns

    def test_matcher_count(self):
        ir = xpath_query().to_ir()
        assert len(ir["matchers"]) == 5


class TestLdapQuery:
    """Tests for ldap_query() sink."""

    def test_returns_or_matcher(self):
        ir = ldap_query().to_ir()
        assert ir["type"] == "logic_or"

    def test_contains_ldap_search(self):
        ir = ldap_query().to_ir()
        methods = [m.get("methodName", "") for m in ir["matchers"]]
        assert "search_s" in methods

    def test_uses_strict_fallback(self):
        ir = ldap_query().to_ir()
        for m in ir["matchers"]:
            if m["type"] == "type_constrained_call":
                assert m["fallbackMode"] == "none"

    def test_matcher_count(self):
        ir = ldap_query().to_ir()
        assert len(ir["matchers"]) == 3


class TestFileWrite:
    """Tests for file_write() sink."""

    def test_returns_or_matcher(self):
        ir = file_write().to_ir()
        assert ir["type"] == "logic_or"

    def test_contains_write(self):
        ir = file_write().to_ir()
        call_patterns = []
        for m in ir["matchers"]:
            if m["type"] == "call_matcher":
                call_patterns.extend(m.get("patterns", []))
        assert "write" in call_patterns

    def test_contains_pathlib(self):
        ir = file_write().to_ir()
        types = [m.get("receiverType", "") for m in ir["matchers"]]
        assert "Path" in types

    def test_uses_strict_fallback(self):
        ir = file_write().to_ir()
        for m in ir["matchers"]:
            if m["type"] == "type_constrained_call":
                assert m["fallbackMode"] == "none"

    def test_matcher_count(self):
        ir = file_write().to_ir()
        assert len(ir["matchers"]) == 4


class TestFileOpen:
    """Tests for file_open() sink."""

    def test_returns_or_matcher(self):
        ir = file_open().to_ir()
        assert ir["type"] == "logic_or"

    def test_contains_open(self):
        ir = file_open().to_ir()
        call_patterns = []
        for m in ir["matchers"]:
            if m["type"] == "call_matcher":
                call_patterns.extend(m.get("patterns", []))
        assert "open" in call_patterns

    def test_matcher_count(self):
        ir = file_open().to_ir()
        assert len(ir["matchers"]) == 2


class TestPathOperation:
    """Tests for path_operation() sink."""

    def test_returns_or_matcher(self):
        ir = path_operation().to_ir()
        assert ir["type"] == "logic_or"

    def test_contains_os_remove(self):
        ir = path_operation().to_ir()
        call_patterns = []
        for m in ir["matchers"]:
            if m["type"] == "call_matcher":
                call_patterns.extend(m.get("patterns", []))
        assert "os.remove" in call_patterns

    def test_contains_shutil(self):
        ir = path_operation().to_ir()
        call_patterns = []
        for m in ir["matchers"]:
            if m["type"] == "call_matcher":
                call_patterns.extend(m.get("patterns", []))
        assert "shutil.copy" in call_patterns
        assert "shutil.rmtree" in call_patterns

    def test_matcher_count(self):
        ir = path_operation().to_ir()
        assert len(ir["matchers"]) == 9


class TestHttpRequest:
    """Tests for http_request() sink."""

    def test_returns_or_matcher(self):
        ir = http_request().to_ir()
        assert ir["type"] == "logic_or"

    def test_contains_requests(self):
        ir = http_request().to_ir()
        call_patterns = []
        for m in ir["matchers"]:
            if m["type"] == "call_matcher":
                call_patterns.extend(m.get("patterns", []))
        assert "requests.get" in call_patterns
        assert "requests.post" in call_patterns

    def test_contains_urllib(self):
        ir = http_request().to_ir()
        call_patterns = []
        for m in ir["matchers"]:
            if m["type"] == "call_matcher":
                call_patterns.extend(m.get("patterns", []))
        assert "urllib.request.urlopen" in call_patterns

    def test_matcher_count(self):
        ir = http_request().to_ir()
        assert len(ir["matchers"]) == 9


class TestSocketConnect:
    """Tests for socket_connect() sink."""

    def test_returns_or_matcher(self):
        ir = socket_connect().to_ir()
        assert ir["type"] == "logic_or"

    def test_contains_connect(self):
        ir = socket_connect().to_ir()
        methods = [m.get("methodName", "") for m in ir["matchers"]]
        assert "connect" in methods

    def test_uses_strict_fallback(self):
        ir = socket_connect().to_ir()
        for m in ir["matchers"]:
            if m["type"] == "type_constrained_call":
                assert m["fallbackMode"] == "none"

    def test_matcher_count(self):
        ir = socket_connect().to_ir()
        assert len(ir["matchers"]) == 2


class TestDeserialize:
    """Tests for deserialize() sink."""

    def test_returns_or_matcher(self):
        ir = deserialize().to_ir()
        assert ir["type"] == "logic_or"

    def test_contains_pickle(self):
        ir = deserialize().to_ir()
        call_patterns = []
        for m in ir["matchers"]:
            if m["type"] == "call_matcher":
                call_patterns.extend(m.get("patterns", []))
        assert "pickle.loads" in call_patterns
        assert "pickle.load" in call_patterns

    def test_contains_yaml(self):
        ir = deserialize().to_ir()
        call_patterns = []
        for m in ir["matchers"]:
            if m["type"] == "call_matcher":
                call_patterns.extend(m.get("patterns", []))
        assert "yaml.load" in call_patterns

    def test_contains_jsonpickle(self):
        ir = deserialize().to_ir()
        call_patterns = []
        for m in ir["matchers"]:
            if m["type"] == "call_matcher":
                call_patterns.extend(m.get("patterns", []))
        assert "jsonpickle.decode" in call_patterns

    def test_matcher_count(self):
        ir = deserialize().to_ir()
        assert len(ir["matchers"]) == 7


class TestSinksInFlows:
    """Tests for using sinks in flows()."""

    def test_sql_execution_as_sink(self):
        m = flows(
            from_sources=sources.http_input(),
            to_sinks=sinks.sql_execution(),
        )
        ir = m.to_ir()
        assert len(ir["sinks"]) == 1
        assert ir["sinks"][0]["type"] == "logic_or"

    def test_command_execution_as_sink(self):
        m = flows(
            from_sources=sources.user_input(),
            to_sinks=sinks.command_execution(),
        )
        ir = m.to_ir()
        assert ir["sinks"][0]["type"] == "logic_or"

    def test_multiple_sinks_in_list(self):
        m = flows(
            from_sources=sources.http_input(),
            to_sinks=[sinks.sql_execution(), sinks.command_execution()],
        )
        ir = m.to_ir()
        assert len(ir["sinks"]) == 2

    def test_full_vulnerability_pattern(self):
        """Simulate what PR-08 vulnerability.sql_injection() will do."""
        m = flows(
            from_sources=sources.http_input(),
            to_sinks=sinks.sql_execution(),
            sanitized_by=calls("parameterize"),
        )
        ir = m.to_ir()
        assert ir["type"] == "dataflow"
        assert len(ir["sources"]) == 1
        assert len(ir["sinks"]) == 1
        assert len(ir["sanitizers"]) == 1


class TestSinksIRRoundTrip:
    """Test JSON round-trip for sink matchers."""

    def test_sql_execution_round_trip(self):
        ir = sql_execution().to_ir()
        json_str = json.dumps(ir)
        parsed = json.loads(json_str)
        assert parsed == ir

    def test_command_execution_round_trip(self):
        ir = command_execution().to_ir()
        json_str = json.dumps(ir)
        parsed = json.loads(json_str)
        assert parsed == ir

    def test_deserialize_round_trip(self):
        ir = deserialize().to_ir()
        json_str = json.dumps(ir)
        parsed = json.loads(json_str)
        assert parsed == ir


class TestSinksStrictMode:
    """Test that type-constrained sinks use fallback='none' for strict matching."""

    def test_sql_execution_strict(self):
        ir = sql_execution().to_ir()
        for m in ir["matchers"]:
            if m["type"] == "type_constrained_call":
                assert m["fallbackMode"] == "none"

    def test_ldap_query_strict(self):
        ir = ldap_query().to_ir()
        for m in ir["matchers"]:
            if m["type"] == "type_constrained_call":
                assert m["fallbackMode"] == "none"

    def test_socket_connect_strict(self):
        ir = socket_connect().to_ir()
        for m in ir["matchers"]:
            if m["type"] == "type_constrained_call":
                assert m["fallbackMode"] == "none"

    def test_template_render_strict(self):
        ir = template_render().to_ir()
        for m in ir["matchers"]:
            if m["type"] == "type_constrained_call":
                assert m["fallbackMode"] == "none"

    def test_file_write_strict(self):
        ir = file_write().to_ir()
        for m in ir["matchers"]:
            if m["type"] == "type_constrained_call":
                assert m["fallbackMode"] == "none"
