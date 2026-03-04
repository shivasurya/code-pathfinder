"""Tests for sources module — semantic source categories."""

import json

import pytest
from codepathfinder import sources, flows, calls
from codepathfinder.sources import (
    http_input,
    http_params,
    http_body,
    http_headers,
    http_cookies,
    file_read,
    file_path,
    env_vars,
    cli_args,
    database_result,
    user_input,
)


class TestHttpParams:
    """Tests for http_params() source."""

    def test_returns_or_matcher(self):
        ir = http_params().to_ir()
        assert ir["type"] == "logic_or"

    def test_contains_django_patterns(self):
        ir = http_params().to_ir()
        types = [m.get("receiverType", "") for m in ir["matchers"]]
        assert "HttpRequest" in types

    def test_contains_flask_patterns(self):
        ir = http_params().to_ir()
        types = [m.get("receiverType", "") for m in ir["matchers"]]
        assert "Request" in types

    def test_contains_generic_patterns(self):
        ir = http_params().to_ir()
        call_patterns = []
        for m in ir["matchers"]:
            if m["type"] == "call_matcher":
                call_patterns.extend(m.get("patterns", []))
        assert "request.get" in call_patterns

    def test_matcher_count(self):
        ir = http_params().to_ir()
        assert len(ir["matchers"]) == 8


class TestHttpBody:
    """Tests for http_body() source."""

    def test_returns_or_matcher(self):
        ir = http_body().to_ir()
        assert ir["type"] == "logic_or"

    def test_contains_json_source(self):
        ir = http_body().to_ir()
        methods = [m.get("methodName", "") for m in ir["matchers"]]
        assert "json" in methods

    def test_contains_get_json(self):
        ir = http_body().to_ir()
        methods = [m.get("methodName", "") for m in ir["matchers"]]
        assert "get_json" in methods

    def test_matcher_count(self):
        ir = http_body().to_ir()
        assert len(ir["matchers"]) == 4


class TestHttpHeaders:
    """Tests for http_headers() source."""

    def test_returns_or_matcher(self):
        ir = http_headers().to_ir()
        assert ir["type"] == "logic_or"

    def test_contains_meta(self):
        ir = http_headers().to_ir()
        methods = [m.get("methodName", "") for m in ir["matchers"]]
        assert "META" in methods

    def test_contains_headers(self):
        ir = http_headers().to_ir()
        methods = [m.get("methodName", "") for m in ir["matchers"]]
        assert "headers" in methods

    def test_matcher_count(self):
        ir = http_headers().to_ir()
        assert len(ir["matchers"]) == 3


class TestHttpCookies:
    """Tests for http_cookies() source."""

    def test_returns_or_matcher(self):
        ir = http_cookies().to_ir()
        assert ir["type"] == "logic_or"

    def test_contains_cookies(self):
        ir = http_cookies().to_ir()
        methods = [m.get("methodName", "") for m in ir["matchers"]]
        assert "COOKIES" in methods
        assert "cookies" in methods

    def test_matcher_count(self):
        ir = http_cookies().to_ir()
        assert len(ir["matchers"]) == 2


class TestHttpInput:
    """Tests for http_input() composite source."""

    def test_returns_or_matcher(self):
        ir = http_input().to_ir()
        assert ir["type"] == "logic_or"

    def test_contains_four_sub_matchers(self):
        ir = http_input().to_ir()
        assert len(ir["matchers"]) == 4

    def test_sub_matchers_are_all_or(self):
        ir = http_input().to_ir()
        for sub in ir["matchers"]:
            assert sub["type"] == "logic_or"


class TestFileRead:
    """Tests for file_read() source."""

    def test_returns_or_matcher(self):
        ir = file_read().to_ir()
        assert ir["type"] == "logic_or"

    def test_contains_open(self):
        ir = file_read().to_ir()
        call_patterns = []
        for m in ir["matchers"]:
            if m["type"] == "call_matcher":
                call_patterns.extend(m.get("patterns", []))
        assert "open" in call_patterns

    def test_contains_pathlib(self):
        ir = file_read().to_ir()
        types = [m.get("receiverType", "") for m in ir["matchers"]]
        assert "Path" in types

    def test_matcher_count(self):
        ir = file_read().to_ir()
        assert len(ir["matchers"]) == 6


class TestFilePath:
    """Tests for file_path() source."""

    def test_returns_or_matcher(self):
        ir = file_path().to_ir()
        assert ir["type"] == "logic_or"

    def test_contains_os_path(self):
        ir = file_path().to_ir()
        call_patterns = []
        for m in ir["matchers"]:
            if m["type"] == "call_matcher":
                call_patterns.extend(m.get("patterns", []))
        assert "os.path.join" in call_patterns
        assert "os.path.abspath" in call_patterns

    def test_matcher_count(self):
        ir = file_path().to_ir()
        assert len(ir["matchers"]) == 4


class TestEnvVars:
    """Tests for env_vars() source."""

    def test_returns_or_matcher(self):
        ir = env_vars().to_ir()
        assert ir["type"] == "logic_or"

    def test_contains_getenv(self):
        ir = env_vars().to_ir()
        call_patterns = []
        for m in ir["matchers"]:
            if m["type"] == "call_matcher":
                call_patterns.extend(m.get("patterns", []))
        assert "os.getenv" in call_patterns

    def test_matcher_count(self):
        ir = env_vars().to_ir()
        assert len(ir["matchers"]) == 3


class TestCliArgs:
    """Tests for cli_args() source."""

    def test_returns_or_matcher(self):
        ir = cli_args().to_ir()
        assert ir["type"] == "logic_or"

    def test_contains_sys_argv(self):
        ir = cli_args().to_ir()
        call_patterns = []
        for m in ir["matchers"]:
            if m["type"] == "call_matcher":
                call_patterns.extend(m.get("patterns", []))
        assert "sys.argv" in call_patterns

    def test_contains_argparse(self):
        ir = cli_args().to_ir()
        types = [m.get("receiverType", "") for m in ir["matchers"]]
        assert "ArgumentParser" in types

    def test_matcher_count(self):
        ir = cli_args().to_ir()
        assert len(ir["matchers"]) == 3


class TestDatabaseResult:
    """Tests for database_result() source."""

    def test_returns_or_matcher(self):
        ir = database_result().to_ir()
        assert ir["type"] == "logic_or"

    def test_contains_cursor_fetch(self):
        ir = database_result().to_ir()
        methods = [m.get("methodName", "") for m in ir["matchers"]]
        assert "fetchone" in methods
        assert "fetchall" in methods
        assert "fetchmany" in methods

    def test_contains_sqlalchemy(self):
        ir = database_result().to_ir()
        types = [m.get("receiverType", "") for m in ir["matchers"]]
        assert "Query" in types

    def test_matcher_count(self):
        ir = database_result().to_ir()
        assert len(ir["matchers"]) == 6


class TestUserInput:
    """Tests for user_input() comprehensive source."""

    def test_returns_or_matcher(self):
        ir = user_input().to_ir()
        assert ir["type"] == "logic_or"

    def test_contains_four_sub_matchers(self):
        ir = user_input().to_ir()
        assert len(ir["matchers"]) == 4

    def test_sub_matchers_are_composites(self):
        ir = user_input().to_ir()
        for sub in ir["matchers"]:
            assert sub["type"] == "logic_or"


class TestSourcesInFlows:
    """Tests for using sources in flows()."""

    def test_http_input_as_source(self):
        m = flows(
            from_sources=sources.http_input(),
            to_sinks=calls("eval"),
        )
        ir = m.to_ir()
        assert len(ir["sources"]) == 1
        assert ir["sources"][0]["type"] == "logic_or"

    def test_http_params_as_source(self):
        m = flows(
            from_sources=sources.http_params(),
            to_sinks=calls("eval"),
        )
        ir = m.to_ir()
        assert ir["sources"][0]["type"] == "logic_or"

    def test_user_input_as_source(self):
        m = flows(
            from_sources=sources.user_input(),
            to_sinks=calls("eval"),
        )
        ir = m.to_ir()
        assert ir["sources"][0]["type"] == "logic_or"

    def test_multiple_sources_in_list(self):
        m = flows(
            from_sources=[sources.http_params(), sources.env_vars()],
            to_sinks=calls("eval"),
        )
        ir = m.to_ir()
        assert len(ir["sources"]) == 2


class TestSourcesIRRoundTrip:
    """Test JSON round-trip for source matchers."""

    def test_http_input_round_trip(self):
        ir = http_input().to_ir()
        json_str = json.dumps(ir)
        parsed = json.loads(json_str)
        assert parsed == ir

    def test_user_input_round_trip(self):
        ir = user_input().to_ir()
        json_str = json.dumps(ir)
        parsed = json.loads(json_str)
        assert parsed == ir

    def test_database_result_round_trip(self):
        ir = database_result().to_ir()
        json_str = json.dumps(ir)
        parsed = json.loads(json_str)
        assert parsed == ir


class TestSourcesFallbackMode:
    """Test that sources use fallback='name' for broad matching."""

    def test_http_params_uses_name_fallback(self):
        ir = http_params().to_ir()
        for m in ir["matchers"]:
            if m["type"] == "type_constrained_call":
                assert m["fallbackMode"] == "name"

    def test_database_result_uses_name_fallback(self):
        ir = database_result().to_ir()
        for m in ir["matchers"]:
            if m["type"] == "type_constrained_call":
                assert m["fallbackMode"] == "name"

    def test_cli_args_uses_name_fallback(self):
        ir = cli_args().to_ir()
        for m in ir["matchers"]:
            if m["type"] == "type_constrained_call":
                assert m["fallbackMode"] == "name"
