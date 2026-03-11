"""Integration tests for QueryType → IR generation pipeline.

Tests that QueryType subclasses produce correct IR JSON that the Go engine
would consume. Validates the full path: Python DSL → IR → JSON.
"""

import json

from codepathfinder import QueryType, gt, regex, missing
from codepathfinder.dataflow import flows
from codepathfinder.logic import Or, And
from codepathfinder.ir import IRType

# --- QueryType definitions (as a user would write them) ---


class DBCursor(QueryType):
    fqns = ["sqlite3.Cursor", "mysql.connector.cursor.MySQLCursor"]
    patterns = ["*Cursor"]
    match_subclasses = True


class WebRequest(QueryType):
    fqns = ["flask.Request", "django.http.HttpRequest"]
    match_subclasses = True


class Hashlib(QueryType):
    fqns = ["hashlib"]


class OSModule(QueryType):
    fqns = ["os"]


# --- Integration tests ---


class TestSQLInjectionRule:
    """Tests the SQL injection rule pattern end-to-end."""

    def test_sql_injection_ir_generation(self):
        """flows(source=WebRequest.method("get"), sink=DBCursor.method("execute"))"""
        result = flows(
            from_sources=WebRequest.method("get"),
            to_sinks=DBCursor.method("execute"),
        )
        ir = result.to_ir()

        assert ir["type"] == IRType.DATAFLOW.value

        # Verify source IR.
        assert len(ir["sources"]) == 1
        source = ir["sources"][0]
        assert source["type"] == IRType.TYPE_CONSTRAINED_CALL.value
        assert source["receiverTypes"] == ["flask.Request", "django.http.HttpRequest"]
        assert source["methodNames"] == ["get"]
        assert source["matchSubclasses"] is True

        # Verify sink IR.
        assert len(ir["sinks"]) == 1
        sink = ir["sinks"][0]
        assert sink["type"] == IRType.TYPE_CONSTRAINED_CALL.value
        assert sink["receiverTypes"] == [
            "sqlite3.Cursor",
            "mysql.connector.cursor.MySQLCursor",
        ]
        assert sink["receiverPatterns"] == ["*Cursor"]
        assert sink["methodNames"] == ["execute"]

    def test_sql_injection_with_sanitizer(self):
        """flows() with a sanitizer."""
        result = flows(
            from_sources=WebRequest.method("get"),
            to_sinks=DBCursor.method("execute"),
            sanitized_by=WebRequest.method("escape"),
        )
        ir = result.to_ir()

        assert len(ir["sanitizers"]) == 1
        san = ir["sanitizers"][0]
        assert san["type"] == IRType.TYPE_CONSTRAINED_CALL.value
        assert san["methodNames"] == ["escape"]

    def test_sql_injection_json_roundtrip(self):
        """Verify IR can be serialized to JSON and parsed back."""
        result = flows(
            from_sources=WebRequest.method("get"),
            to_sinks=DBCursor.method("execute"),
        )
        ir = result.to_ir()

        json_str = json.dumps(ir)
        parsed = json.loads(json_str)

        assert parsed["type"] == "dataflow"
        assert len(parsed["sources"]) == 1
        assert len(parsed["sinks"]) == 1
        assert parsed["sources"][0]["type"] == "type_constrained_call"


class TestWeakHashRule:
    """Tests the weak hash detection pattern with Or()."""

    def test_or_weak_hash_ir(self):
        """Or(Hashlib.method("md5"), Hashlib.method("sha1"))"""
        result = Or(Hashlib.method("md5"), Hashlib.method("sha1"))
        ir = result.to_ir()

        assert ir["type"] == IRType.LOGIC_OR.value
        assert len(ir["matchers"]) == 2

        md5_ir = ir["matchers"][0]
        assert md5_ir["type"] == IRType.TYPE_CONSTRAINED_CALL.value
        assert md5_ir["receiverTypes"] == ["hashlib"]
        assert md5_ir["methodNames"] == ["md5"]

        sha1_ir = ir["matchers"][1]
        assert sha1_ir["methodNames"] == ["sha1"]

    def test_or_json_roundtrip(self):
        result = Or(Hashlib.method("md5"), Hashlib.method("sha1"))
        ir = result.to_ir()
        json_str = json.dumps(ir)
        parsed = json.loads(json_str)

        assert parsed["type"] == "logic_or"
        assert len(parsed["matchers"]) == 2


class TestArgumentConstraints:
    """Tests argument matching with qualifiers in integration context."""

    def test_chmod_wildcard_arg(self):
        """OSModule.method("chmod").arg(1, "0o7*")"""
        matcher = OSModule.method("chmod").arg(1, "0o7*")
        ir = matcher.to_ir()

        assert ir["positionalArgs"]["1"]["value"] == "0o7*"
        assert ir["positionalArgs"]["1"]["wildcard"] is True

    def test_chmod_comparator_arg(self):
        """OSModule.method("chmod").arg(1, gt(0o644))"""
        matcher = OSModule.method("chmod").arg(1, gt(0o644))
        ir = matcher.to_ir()

        assert ir["positionalArgs"]["1"]["comparator"] == "gt"
        assert ir["positionalArgs"]["1"]["value"] == 0o644

    def test_missing_keyword(self):
        """DBCursor.method("execute").arg("timeout", missing())"""
        matcher = DBCursor.method("execute").arg("timeout", missing())
        ir = matcher.to_ir()

        assert ir["keywordArgs"]["timeout"]["comparator"] == "missing"

    def test_regex_arg(self):
        """WebRequest.method("get").arg(0, regex("^/api/.*"))"""
        matcher = WebRequest.method("get").arg(0, regex("^/api/.*"))
        ir = matcher.to_ir()

        assert ir["positionalArgs"]["0"]["comparator"] == "regex"
        assert ir["positionalArgs"]["0"]["value"] == "^/api/.*"


class TestComplexRuleComposition:
    """Tests complex rule patterns combining multiple features."""

    def test_and_logic_with_typed_matchers(self):
        """And(DBCursor.method("connect"), DBCursor.method("execute"))"""
        result = And(DBCursor.method("connect"), DBCursor.method("execute"))
        ir = result.to_ir()

        assert ir["type"] == IRType.LOGIC_AND.value
        assert len(ir["matchers"]) == 2

    def test_dataflow_with_or_source(self):
        """flows(source=Or(WebRequest.method("get"), WebRequest.method("form")), sink=...)"""
        result = flows(
            from_sources=Or(WebRequest.method("get"), WebRequest.method("form")),
            to_sinks=DBCursor.method("execute"),
        )
        ir = result.to_ir()

        assert ir["type"] == "dataflow"
        # Source should be the Or IR.
        assert len(ir["sources"]) == 1
        source = ir["sources"][0]
        assert source["type"] == "logic_or"

    def test_multi_method_matcher(self):
        """DBCursor.method("execute", "executemany", "executescript")"""
        matcher = DBCursor.method("execute", "executemany", "executescript")
        ir = matcher.to_ir()

        assert ir["methodNames"] == ["execute", "executemany", "executescript"]
        assert ir["receiverTypes"] == [
            "sqlite3.Cursor",
            "mysql.connector.cursor.MySQLCursor",
        ]
        assert ir["receiverPatterns"] == ["*Cursor"]

    def test_chained_arg_constraints(self):
        """Multiple .arg() calls chain correctly."""
        matcher = OSModule.method("chmod").arg(0, "/tmp/*").arg(1, gt(0o644))
        ir = matcher.to_ir()

        assert "0" in ir["positionalArgs"]
        assert "1" in ir["positionalArgs"]
        assert ir["positionalArgs"]["0"]["value"] == "/tmp/*"
        assert ir["positionalArgs"]["0"]["wildcard"] is True
        assert ir["positionalArgs"]["1"]["comparator"] == "gt"
