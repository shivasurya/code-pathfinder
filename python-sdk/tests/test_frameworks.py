"""Tests for framework presets — Django, Flask, FastAPI."""

import json

import pytest
from codepathfinder import flows
from codepathfinder.frameworks import django, flask, fastapi


class TestDjangoSources:
    def test_request_data_returns_or(self):
        ir = django.sources.request_data().to_ir()
        assert ir["type"] == "logic_or"

    def test_request_data_has_get_post(self):
        ir = django.sources.request_data().to_ir()
        types = [m.get("receiverType", "") for m in ir["matchers"]]
        assert "HttpRequest" in types
        assert "QueryDict" in types

    def test_request_data_matcher_count(self):
        ir = django.sources.request_data().to_ir()
        assert len(ir["matchers"]) == 6

    def test_session_data(self):
        ir = django.sources.session_data().to_ir()
        assert ir["type"] == "type_constrained_call"
        assert ir["receiverType"] == "SessionBase"

    def test_url_params(self):
        ir = django.sources.url_params().to_ir()
        assert ir["type"] == "call_matcher"


class TestDjangoSinks:
    def test_raw_sql_returns_or(self):
        ir = django.sinks.raw_sql().to_ir()
        assert ir["type"] == "logic_or"

    def test_raw_sql_has_cursor_execute(self):
        ir = django.sinks.raw_sql().to_ir()
        methods = [m.get("methodName", "") for m in ir["matchers"]]
        assert "execute" in methods

    def test_raw_sql_strict_fallback(self):
        ir = django.sinks.raw_sql().to_ir()
        for m in ir["matchers"]:
            if m["type"] == "type_constrained_call":
                assert m["fallbackMode"] == "none", (
                    f"{m['receiverType']}.{m['methodName']} must use fallback='none'"
                )

    def test_template_render(self):
        ir = django.sinks.template_render().to_ir()
        assert ir["type"] == "logic_or"
        assert len(ir["matchers"]) == 3

    def test_redirect(self):
        ir = django.sinks.redirect().to_ir()
        assert ir["type"] == "call_matcher"
        assert "HttpResponseRedirect" in ir["patterns"]


class TestDjangoSanitizers:
    def test_orm_parameterization(self):
        ir = django.sanitizers.orm_parameterization().to_ir()
        assert ir["type"] == "type_constrained_call"
        assert ir["receiverType"] == "QuerySet"
        assert ir["methodName"] == "filter"

    def test_escape(self):
        ir = django.sanitizers.escape().to_ir()
        assert ir["type"] == "call_matcher"
        assert "django.utils.html.escape" in ir["patterns"]


class TestFlaskSources:
    def test_request_data_returns_or(self):
        ir = flask.sources.request_data().to_ir()
        assert ir["type"] == "logic_or"

    def test_request_data_has_all_fields(self):
        ir = flask.sources.request_data().to_ir()
        methods = [m.get("methodName", "") for m in ir["matchers"]]
        assert "args" in methods
        assert "form" in methods
        assert "json" in methods
        assert "files" in methods

    def test_request_data_matcher_count(self):
        ir = flask.sources.request_data().to_ir()
        assert len(ir["matchers"]) == 6

    def test_headers(self):
        ir = flask.sources.headers().to_ir()
        assert ir["type"] == "type_constrained_call"
        assert ir["methodName"] == "headers"

    def test_cookies(self):
        ir = flask.sources.cookies().to_ir()
        assert ir["type"] == "type_constrained_call"
        assert ir["methodName"] == "cookies"


class TestFlaskSinks:
    def test_render_template(self):
        ir = flask.sinks.render_template().to_ir()
        assert ir["type"] == "logic_or"
        assert len(ir["matchers"]) == 2

    def test_send_file(self):
        ir = flask.sinks.send_file().to_ir()
        assert ir["type"] == "call_matcher"
        assert "send_file" in ir["patterns"]

    def test_redirect(self):
        ir = flask.sinks.redirect().to_ir()
        assert ir["type"] == "call_matcher"
        assert "redirect" in ir["patterns"]


class TestFlaskSanitizers:
    def test_escape(self):
        ir = flask.sanitizers.escape().to_ir()
        assert ir["type"] == "call_matcher"
        assert "markupsafe.escape" in ir["patterns"]

    def test_autoescape(self):
        ir = flask.sanitizers.autoescape().to_ir()
        assert ir["type"] == "call_matcher"


class TestFastAPISources:
    def test_request_params(self):
        ir = fastapi.sources.request_params().to_ir()
        assert ir["type"] == "logic_or"
        assert len(ir["matchers"]) == 5

    def test_request_params_has_query_path_body(self):
        ir = fastapi.sources.request_params().to_ir()
        patterns = []
        for m in ir["matchers"]:
            patterns.extend(m.get("patterns", []))
        assert "Query" in patterns
        assert "Path" in patterns
        assert "Body" in patterns

    def test_request_data(self):
        ir = fastapi.sources.request_data().to_ir()
        assert ir["type"] == "logic_or"
        assert len(ir["matchers"]) == 3

    def test_form_data(self):
        ir = fastapi.sources.form_data().to_ir()
        assert ir["type"] == "call_matcher"
        assert "Form" in ir["patterns"]


class TestFastAPISinks:
    def test_json_response(self):
        ir = fastapi.sinks.json_response().to_ir()
        assert ir["type"] == "logic_or"

    def test_redirect(self):
        ir = fastapi.sinks.redirect().to_ir()
        assert ir["type"] == "call_matcher"
        assert "RedirectResponse" in ir["patterns"]

    def test_file_response(self):
        ir = fastapi.sinks.file_response().to_ir()
        assert ir["type"] == "call_matcher"
        assert "FileResponse" in ir["patterns"]


class TestFastAPISanitizers:
    def test_pydantic_validation(self):
        ir = fastapi.sanitizers.pydantic_validation().to_ir()
        assert ir["type"] == "type_constrained_call"
        assert ir["receiverType"] == "BaseModel"

    def test_depends(self):
        ir = fastapi.sanitizers.depends().to_ir()
        assert ir["type"] == "call_matcher"
        assert "Depends" in ir["patterns"]


class TestFrameworkInFlows:
    """Test framework presets work inside flows()."""

    def test_django_sql_injection_flow(self):
        result = flows(
            from_sources=django.sources.request_data(),
            to_sinks=django.sinks.raw_sql(),
            sanitized_by=django.sanitizers.orm_parameterization(),
        )
        ir = result.to_ir()
        assert ir["type"] == "dataflow"
        assert ir["scope"] == "global"
        assert len(ir["sources"]) == 1
        assert len(ir["sinks"]) == 1
        assert len(ir["sanitizers"]) == 1

    def test_flask_ssti_flow(self):
        result = flows(
            from_sources=flask.sources.request_data(),
            to_sinks=flask.sinks.render_template(),
            sanitized_by=flask.sanitizers.escape(),
        )
        ir = result.to_ir()
        assert ir["type"] == "dataflow"

    def test_fastapi_flow(self):
        result = flows(
            from_sources=fastapi.sources.request_params(),
            to_sinks=fastapi.sinks.json_response(),
        )
        ir = result.to_ir()
        assert ir["type"] == "dataflow"

    def test_cross_framework_flow(self):
        """Framework sources can be mixed with generic sinks."""
        from codepathfinder import sinks as generic_sinks

        result = flows(
            from_sources=django.sources.request_data(),
            to_sinks=generic_sinks.sql_execution(),
        )
        ir = result.to_ir()
        assert ir["type"] == "dataflow"


class TestFrameworkImportStyle:
    """Test import patterns work correctly."""

    def test_import_django(self):
        from codepathfinder.frameworks import django as dj

        assert hasattr(dj, "sources")
        assert hasattr(dj, "sinks")
        assert hasattr(dj, "sanitizers")

    def test_import_flask(self):
        from codepathfinder.frameworks import flask as fl

        assert hasattr(fl, "sources")
        assert hasattr(fl, "sinks")
        assert hasattr(fl, "sanitizers")

    def test_import_fastapi(self):
        from codepathfinder.frameworks import fastapi as fa

        assert hasattr(fa, "sources")
        assert hasattr(fa, "sinks")
        assert hasattr(fa, "sanitizers")

    def test_import_frameworks_package(self):
        from codepathfinder import frameworks

        assert hasattr(frameworks, "django")
        assert hasattr(frameworks, "flask")
        assert hasattr(frameworks, "fastapi")


class TestAllFrameworksIRValid:
    """Loop all framework functions and validate they produce valid IR."""

    FRAMEWORK_FUNCTIONS = [
        ("django.sources.request_data", django.sources.request_data),
        ("django.sources.session_data", django.sources.session_data),
        ("django.sources.url_params", django.sources.url_params),
        ("django.sinks.raw_sql", django.sinks.raw_sql),
        ("django.sinks.template_render", django.sinks.template_render),
        ("django.sinks.redirect", django.sinks.redirect),
        ("django.sanitizers.orm_parameterization", django.sanitizers.orm_parameterization),
        ("django.sanitizers.escape", django.sanitizers.escape),
        ("flask.sources.request_data", flask.sources.request_data),
        ("flask.sources.headers", flask.sources.headers),
        ("flask.sources.cookies", flask.sources.cookies),
        ("flask.sinks.render_template", flask.sinks.render_template),
        ("flask.sinks.send_file", flask.sinks.send_file),
        ("flask.sinks.redirect", flask.sinks.redirect),
        ("flask.sanitizers.escape", flask.sanitizers.escape),
        ("flask.sanitizers.autoescape", flask.sanitizers.autoescape),
        ("fastapi.sources.request_params", fastapi.sources.request_params),
        ("fastapi.sources.request_data", fastapi.sources.request_data),
        ("fastapi.sources.form_data", fastapi.sources.form_data),
        ("fastapi.sinks.json_response", fastapi.sinks.json_response),
        ("fastapi.sinks.redirect", fastapi.sinks.redirect),
        ("fastapi.sinks.file_response", fastapi.sinks.file_response),
        ("fastapi.sanitizers.pydantic_validation", fastapi.sanitizers.pydantic_validation),
        ("fastapi.sanitizers.depends", fastapi.sanitizers.depends),
    ]

    @pytest.mark.parametrize(
        "name,fn",
        FRAMEWORK_FUNCTIONS,
        ids=lambda x: x if isinstance(x, str) else "",
    )
    def test_produces_valid_ir(self, name, fn):
        ir = fn().to_ir()
        assert "type" in ir, f"{name} IR missing 'type'"

    @pytest.mark.parametrize(
        "name,fn",
        FRAMEWORK_FUNCTIONS,
        ids=lambda x: x if isinstance(x, str) else "",
    )
    def test_json_serializable(self, name, fn):
        ir = fn().to_ir()
        serialized = json.dumps(ir)
        assert len(serialized) > 0
