"""Tests for QueryType, MethodMatcher, and qualifiers."""

import pytest
from codepathfinder import QueryType, Or, And, flows, calls
from codepathfinder.qualifiers import lt, gt, lte, gte, regex, missing
from codepathfinder.query_type import MethodMatcher

# --- Test QueryType subclasses ---


class DBCursor(QueryType):
    fqns = ["sqlite3.Cursor", "psycopg2.extensions.cursor"]
    patterns = ["*.Cursor"]


class Subprocess(QueryType):
    fqns = ["subprocess"]


class Hashlib(QueryType):
    fqns = ["hashlib"]


class RSA(QueryType):
    fqns = ["cryptography.hazmat.primitives.asymmetric.rsa"]


class Requests(QueryType):
    fqns = ["requests", "httpx"]
    patterns = []
    match_subclasses = False


class WebRequest(QueryType):
    fqns = ["flask.request", "django.http.HttpRequest"]
    patterns = ["*.Request"]


class Response(QueryType):
    fqns = ["flask.Response", "django.http.HttpResponse"]
    patterns = ["*.Response"]


# --- QueryType.method() tests ---


def test_method_returns_method_matcher():
    matcher = DBCursor.method("execute")
    assert isinstance(matcher, MethodMatcher)


def test_method_single():
    ir = DBCursor.method("execute").to_ir()
    assert ir["type"] == "type_constrained_call"
    assert ir["methodNames"] == ["execute"]
    assert ir["receiverTypes"] == ["sqlite3.Cursor", "psycopg2.extensions.cursor"]
    assert ir["receiverPatterns"] == ["*.Cursor"]
    assert ir["matchSubclasses"] is True


def test_method_multiple():
    ir = Subprocess.method("call", "run", "Popen").to_ir()
    assert ir["methodNames"] == ["call", "run", "Popen"]
    assert ir["receiverTypes"] == ["subprocess"]


def test_method_no_args_raises():
    with pytest.raises(ValueError):
        DBCursor.method()


def test_match_subclasses_default_true():
    ir = DBCursor.method("execute").to_ir()
    assert ir["matchSubclasses"] is True


def test_match_subclasses_false():
    ir = Requests.method("get").to_ir()
    assert ir["matchSubclasses"] is False


def test_min_confidence_default():
    ir = DBCursor.method("execute").to_ir()
    assert ir["minConfidence"] == 0.5


def test_fallback_mode_default():
    ir = DBCursor.method("execute").to_ir()
    assert ir["fallbackMode"] == "none"


# --- .arg() chaining tests ---


def test_arg_positional():
    ir = DBCursor.method("execute").arg(0, "SELECT 1").to_ir()
    assert "positionalArgs" in ir
    assert ir["positionalArgs"]["0"]["value"] == "SELECT 1"
    assert ir["positionalArgs"]["0"]["wildcard"] is False


def test_arg_keyword_bool():
    ir = Subprocess.method("call").arg("shell", True).to_ir()
    assert "keywordArgs" in ir
    assert ir["keywordArgs"]["shell"]["value"] is True
    assert ir["keywordArgs"]["shell"]["wildcard"] is False


def test_arg_chaining():
    ir = (
        Response.method("set_cookie")
        .arg("secure", missing())
        .arg("httponly", missing())
        .to_ir()
    )
    assert "keywordArgs" in ir
    assert ir["keywordArgs"]["secure"]["comparator"] == "missing"
    assert ir["keywordArgs"]["httponly"]["comparator"] == "missing"


def test_arg_no_positional_or_keyword_omitted():
    ir = DBCursor.method("execute").to_ir()
    assert "positionalArgs" not in ir
    assert "keywordArgs" not in ir


def test_arg_wildcard_string():
    ir = Requests.method("get").arg(0, "http://*").to_ir()
    assert ir["positionalArgs"]["0"]["wildcard"] is True


# --- Qualifier tests ---


def test_qualifier_lt():
    ir = RSA.method("generate_private_key").arg("key_size", lt(2048)).to_ir()
    kw = ir["keywordArgs"]["key_size"]
    assert kw["value"] == 2048
    assert kw["comparator"] == "lt"
    assert kw["wildcard"] is False


def test_qualifier_gt():
    constraint = gt(100).to_constraint()
    assert constraint["value"] == 100
    assert constraint["comparator"] == "gt"


def test_qualifier_lte():
    constraint = lte(1024).to_constraint()
    assert constraint["value"] == 1024
    assert constraint["comparator"] == "lte"


def test_qualifier_gte():
    constraint = gte(256).to_constraint()
    assert constraint["value"] == 256
    assert constraint["comparator"] == "gte"


def test_qualifier_regex():
    ir = Requests.method("get").arg(0, regex(r"http://.*")).to_ir()
    pos = ir["positionalArgs"]["0"]
    assert pos["value"] == "http://.*"
    assert pos["comparator"] == "regex"


def test_qualifier_missing():
    ir = Requests.method("get").arg("timeout", missing()).to_ir()
    kw = ir["keywordArgs"]["timeout"]
    assert kw["value"] is None
    assert kw["comparator"] == "missing"


# --- Logic operator integration ---


def test_or_with_method_matchers():
    result = Or(Hashlib.method("md5"), Hashlib.method("sha1"))
    ir = result.to_ir()
    assert ir["type"] == "logic_or"
    assert len(ir["matchers"]) == 2
    assert ir["matchers"][0]["type"] == "type_constrained_call"
    assert ir["matchers"][0]["methodNames"] == ["md5"]
    assert ir["matchers"][1]["methodNames"] == ["sha1"]


def test_and_with_method_matchers():
    result = And(
        Subprocess.method("call").arg("shell", True),
        Subprocess.method("call").arg(0, regex(r".*rm.*")),
    )
    ir = result.to_ir()
    assert ir["type"] == "logic_and"
    assert len(ir["matchers"]) == 2


def test_or_mixed_matchers():
    result = Or(calls("eval"), Hashlib.method("md5"))
    ir = result.to_ir()
    assert ir["type"] == "logic_or"
    assert ir["matchers"][0]["type"] == "call_matcher"
    assert ir["matchers"][1]["type"] == "type_constrained_call"


# --- flows() integration ---


def test_flows_with_method_matchers():
    result = flows(
        from_sources=WebRequest.method("get", "args"),
        to_sinks=DBCursor.method("execute"),
    )
    ir = result.to_ir()
    assert ir["type"] == "dataflow"
    assert len(ir["sources"]) == 1
    assert ir["sources"][0]["type"] == "type_constrained_call"
    assert len(ir["sinks"]) == 1
    assert ir["sinks"][0]["type"] == "type_constrained_call"


def test_flows_mixed_matchers():
    result = flows(
        from_sources=calls("request.GET"),
        to_sinks=DBCursor.method("execute"),
    )
    ir = result.to_ir()
    assert ir["sources"][0]["type"] == "call_matcher"
    assert ir["sinks"][0]["type"] == "type_constrained_call"


def test_flows_with_sanitizer_method_matcher():
    result = flows(
        from_sources=WebRequest.method("get"),
        to_sinks=DBCursor.method("execute"),
        sanitized_by=DBCursor.method("execute").arg(1, True),
    )
    ir = result.to_ir()
    assert len(ir["sanitizers"]) == 1
    assert ir["sanitizers"][0]["type"] == "type_constrained_call"


# --- repr tests ---


def test_method_matcher_repr():
    matcher = DBCursor.method("execute", "executemany")
    assert "execute" in repr(matcher)
    assert "executemany" in repr(matcher)


# --- Full rule composition example ---


def test_full_sql_injection_rule_ir():
    ir = flows(
        from_sources=WebRequest.method("get", "args", "form"),
        to_sinks=DBCursor.method("execute", "executemany"),
    ).to_ir()

    assert ir["type"] == "dataflow"
    src = ir["sources"][0]
    assert src["receiverTypes"] == ["flask.request", "django.http.HttpRequest"]
    assert src["methodNames"] == ["get", "args", "form"]
    sink = ir["sinks"][0]
    assert sink["receiverTypes"] == ["sqlite3.Cursor", "psycopg2.extensions.cursor"]
    assert sink["methodNames"] == ["execute", "executemany"]
