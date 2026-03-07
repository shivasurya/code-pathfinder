"""Tests for the typeshed stub-to-JSON converter."""

from __future__ import annotations

import ast
import json
import textwrap
from pathlib import Path
from typing import Any

import pytest

from convert import (
    BUILTIN_MAP,
    _extract_reexports,
    _find_inner_package_dir,
    _has_decorator,
    _infer_assign_type,
    _resolve_attribute_fqn,
    _resolve_union,
    _typeshed_importable_name,
    convert_package,
    convert_stub_file,
    extract_class,
    extract_function,
    extract_params,
    generate_manifest,
    load_sources,
    resolve_type_annotation,
)


# ── Fixtures ──────────────────────────────────────────────────────────────────


@pytest.fixture
def tmp_package(tmp_path: Path) -> Path:
    """Create a minimal typeshed-style package directory."""
    pkg = tmp_path / "stubs" / "mypkg" / "mypkg"
    pkg.mkdir(parents=True)
    return pkg


@pytest.fixture
def write_stub(tmp_package: Path):
    """Helper to write a .pyi file into the temp package."""
    def _write(filename: str, content: str) -> Path:
        path = tmp_package / filename
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(textwrap.dedent(content))
        return path
    return _write


# ── resolve_type_annotation tests ─────────────────────────────────────────────


class TestResolveTypeAnnotation:
    """Tests for type annotation resolution."""

    def test_none_annotation(self):
        assert resolve_type_annotation(None, "mod") == "builtins.NoneType"

    def test_none_constant(self):
        node = ast.Constant(value=None)
        assert resolve_type_annotation(node, "mod") == "builtins.NoneType"

    def test_string_constant_forward_ref(self):
        node = ast.Constant(value="Response")
        assert resolve_type_annotation(node, "requests") == "requests.Response"

    def test_string_constant_builtin(self):
        node = ast.Constant(value="str")
        assert resolve_type_annotation(node, "mod") == "builtins.str"

    def test_string_constant_incomplete(self):
        node = ast.Constant(value="Incomplete")
        assert resolve_type_annotation(node, "mod") == "builtins.object"

    def test_numeric_constant(self):
        node = ast.Constant(value=42)
        assert resolve_type_annotation(node, "mod") == "builtins.int"

    @pytest.mark.parametrize("name,expected", [
        ("str", "builtins.str"),
        ("int", "builtins.int"),
        ("float", "builtins.float"),
        ("bool", "builtins.bool"),
        ("bytes", "builtins.bytes"),
        ("list", "builtins.list"),
        ("dict", "builtins.dict"),
        ("set", "builtins.set"),
        ("tuple", "builtins.tuple"),
        ("None", "builtins.NoneType"),
        ("object", "builtins.object"),
        ("type", "builtins.type"),
        ("Exception", "builtins.Exception"),
    ])
    def test_builtin_names(self, name: str, expected: str):
        node = ast.Name(id=name)
        assert resolve_type_annotation(node, "mod") == expected

    def test_opaque_name_incomplete(self):
        node = ast.Name(id="Incomplete")
        assert resolve_type_annotation(node, "mod") == "builtins.object"

    def test_opaque_name_typevar(self):
        node = ast.Name(id="TypeVar")
        assert resolve_type_annotation(node, "mod") == "builtins.object"

    def test_typing_name(self):
        node = ast.Name(id="Any")
        assert resolve_type_annotation(node, "mod") == "typing.Any"

    def test_module_qualified_name(self):
        node = ast.Name(id="Response")
        assert resolve_type_annotation(node, "requests") == "requests.Response"

    def test_attribute_node(self):
        # os.PathLike
        node = ast.Attribute(
            value=ast.Name(id="os"),
            attr="PathLike",
        )
        assert resolve_type_annotation(node, "mod") == "os.PathLike"

    def test_optional_unwrap(self):
        # Optional[str]
        node = ast.Subscript(
            value=ast.Name(id="Optional"),
            slice=ast.Name(id="str"),
        )
        assert resolve_type_annotation(node, "mod") == "builtins.str"

    def test_union_unwrap_first_non_none(self):
        # Union[str, None]
        node = ast.Subscript(
            value=ast.Name(id="Union"),
            slice=ast.Tuple(elts=[
                ast.Name(id="str"),
                ast.Constant(value=None),
            ]),
        )
        assert resolve_type_annotation(node, "mod") == "builtins.str"

    def test_union_all_none(self):
        # Union[None, None]
        node = ast.Subscript(
            value=ast.Name(id="Union"),
            slice=ast.Tuple(elts=[
                ast.Constant(value=None),
                ast.Constant(value=None),
            ]),
        )
        assert resolve_type_annotation(node, "mod") == "builtins.NoneType"

    def test_generic_strips_param(self):
        # list[str] -> builtins.list
        node = ast.Subscript(
            value=ast.Name(id="list"),
            slice=ast.Name(id="str"),
        )
        assert resolve_type_annotation(node, "mod") == "builtins.list"

    def test_pep604_bitor(self):
        # str | None -> builtins.str
        node = ast.BinOp(
            left=ast.Name(id="str"),
            op=ast.BitOr(),
            right=ast.Constant(value=None),
        )
        assert resolve_type_annotation(node, "mod") == "builtins.str"

    def test_pep604_bitor_none_first(self):
        # None | str -> builtins.str
        node = ast.BinOp(
            left=ast.Constant(value=None),
            op=ast.BitOr(),
            right=ast.Name(id="str"),
        )
        assert resolve_type_annotation(node, "mod") == "builtins.str"

    def test_list_node_fallback(self):
        node = ast.List(elts=[])
        assert resolve_type_annotation(node, "mod") == "builtins.object"

    def test_unknown_node_fallback(self):
        node = ast.Starred(value=ast.Name(id="x"))
        assert resolve_type_annotation(node, "mod") == "builtins.object"


class TestResolveUnion:
    def test_single_element(self):
        node = ast.Name(id="str")
        assert _resolve_union(node, "mod") == "builtins.str"

    def test_tuple_elements(self):
        node = ast.Tuple(elts=[
            ast.Constant(value=None),
            ast.Name(id="int"),
        ])
        assert _resolve_union(node, "mod") == "builtins.int"


class TestResolveAttributeFqn:
    def test_simple(self):
        node = ast.Attribute(value=ast.Name(id="os"), attr="path")
        assert _resolve_attribute_fqn(node) == "os.path"

    def test_deep_chain(self):
        node = ast.Attribute(
            value=ast.Attribute(
                value=ast.Name(id="a"),
                attr="b",
            ),
            attr="c",
        )
        assert _resolve_attribute_fqn(node) == "a.b.c"


# ── extract_params tests ──────────────────────────────────────────────────────


class TestExtractParams:
    def _parse_func(self, code: str) -> ast.FunctionDef:
        tree = ast.parse(textwrap.dedent(code))
        return tree.body[0]

    def test_no_params(self):
        func = self._parse_func("def f(): ...")
        params = extract_params(func.args, "mod")
        assert params == []

    def test_self_skipped(self):
        func = self._parse_func("def f(self, x: int): ...")
        params = extract_params(func.args, "mod")
        assert len(params) == 1
        assert params[0]["name"] == "x"

    def test_cls_skipped(self):
        func = self._parse_func("def f(cls, x: int): ...")
        params = extract_params(func.args, "mod")
        assert len(params) == 1

    def test_required_vs_optional(self):
        func = self._parse_func("def f(a: int, b: str = 'x'): ...")
        params = extract_params(func.args, "mod")
        assert params[0]["required"] is True
        assert params[1]["required"] is False

    def test_varargs(self):
        func = self._parse_func("def f(*args: int): ...")
        params = extract_params(func.args, "mod")
        assert params[0]["name"] == "*args"
        assert params[0]["required"] is False

    def test_kwargs(self):
        func = self._parse_func("def f(**kwargs: str): ...")
        params = extract_params(func.args, "mod")
        assert params[0]["name"] == "**kwargs"
        assert params[0]["required"] is False

    def test_kwonly_args(self):
        func = self._parse_func("def f(*, key: str, value: int = 0): ...")
        params = extract_params(func.args, "mod")
        assert len(params) == 2
        assert params[0]["name"] == "key"
        assert params[0]["required"] is True
        assert params[1]["name"] == "value"
        assert params[1]["required"] is False


# ── extract_function tests ────────────────────────────────────────────────────


class TestExtractFunction:
    def _parse_func(self, code: str) -> ast.FunctionDef:
        tree = ast.parse(textwrap.dedent(code))
        return tree.body[0]

    def test_simple_function(self):
        func = self._parse_func("def get(url: str) -> Response: ...")
        result = extract_function(func, "requests", "typeshed")
        assert result["return_type"] == "requests.Response"
        assert result["confidence"] == 0.95
        assert result["source"] == "typeshed"
        assert len(result["params"]) == 1
        assert result["params"][0]["name"] == "url"

    def test_no_return_annotation(self):
        func = self._parse_func("def f(): ...")
        result = extract_function(func, "mod", "typeshed")
        assert result["return_type"] == "builtins.NoneType"


# ── extract_class tests ──────────────────────────────────────────────────────


class TestExtractClass:
    def _parse_class(self, code: str) -> ast.ClassDef:
        tree = ast.parse(textwrap.dedent(code))
        return tree.body[0]

    def test_simple_class(self):
        cls = self._parse_class("""
            class Response:
                status_code: int
                def json(self) -> dict: ...
        """)
        result = extract_class(cls, "requests", "typeshed")
        assert result["type"] == "class"
        assert "json" in result["methods"]
        assert "status_code" in result["attributes"]
        assert result["attributes"]["status_code"]["type"] == "builtins.int"
        assert result["attributes"]["status_code"]["kind"] == "attribute"

    def test_property_extraction(self):
        cls = self._parse_class("""
            class Request:
                @property
                def content(self) -> bytes: ...
        """)
        result = extract_class(cls, "requests", "typeshed")
        assert "content" in result["attributes"]
        assert result["attributes"]["content"]["kind"] == "property"
        assert result["attributes"]["content"]["type"] == "builtins.bytes"
        assert "content" not in result["methods"]

    def test_init_return_type(self):
        cls = self._parse_class("""
            class Session:
                def __init__(self) -> None: ...
        """)
        result = extract_class(cls, "requests", "typeshed")
        assert result["methods"]["__init__"]["return_type"] == "requests.Session"

    def test_base_classes(self):
        cls = self._parse_class("""
            class MyView(View):
                pass
        """)
        result = extract_class(cls, "django.views", "django-stubs")
        assert "django.views.View" in result["bases"]

    def test_object_base_excluded(self):
        cls = self._parse_class("""
            class Foo(object):
                pass
        """)
        result = extract_class(cls, "mod", "typeshed")
        assert result["bases"] == []

    def test_overload_methods(self):
        cls = self._parse_class("""
            class Foo:
                @overload
                def bar(self, x: int) -> int: ...
                @overload
                def bar(self, x: str) -> str: ...
        """)
        result = extract_class(cls, "mod", "typeshed")
        assert "bar" in result["methods"]
        assert result["methods"]["bar"]["return_type"] == "builtins.int"

    def test_overload_with_implementation(self):
        cls = self._parse_class("""
            class Foo:
                @overload
                def bar(self, x: int) -> int: ...
                @overload
                def bar(self, x: str) -> str: ...
                def bar(self, x): ...
        """)
        result = extract_class(cls, "mod", "typeshed")
        # Implementation wins (already in methods dict)
        assert "bar" in result["methods"]


# ── _has_decorator tests ──────────────────────────────────────────────────────


class TestHasDecorator:
    def _parse_func(self, code: str) -> ast.FunctionDef:
        tree = ast.parse(textwrap.dedent(code))
        return tree.body[0]

    def test_name_decorator(self):
        func = self._parse_func("@property\ndef f(): ...")
        assert _has_decorator(func, "property") is True

    def test_attribute_decorator(self):
        func = self._parse_func("@abc.abstractmethod\ndef f(): ...")
        assert _has_decorator(func, "abstractmethod") is True

    def test_no_decorator(self):
        func = self._parse_func("def f(): ...")
        assert _has_decorator(func, "property") is False


# ── _infer_assign_type tests ─────────────────────────────────────────────────


class TestInferAssignType:
    def test_none_value(self):
        assert _infer_assign_type(None, "mod") == "builtins.object"

    def test_int_constant(self):
        node = ast.Constant(value=42)
        assert _infer_assign_type(node, "mod") == "builtins.int"

    def test_str_constant(self):
        node = ast.Constant(value="hello")
        assert _infer_assign_type(node, "mod") == "builtins.str"

    def test_none_constant(self):
        node = ast.Constant(value=None)
        assert _infer_assign_type(node, "mod") == "builtins.NoneType"

    def test_list(self):
        node = ast.List(elts=[])
        assert _infer_assign_type(node, "mod") == "builtins.list"

    def test_dict(self):
        node = ast.Dict(keys=[], values=[])
        assert _infer_assign_type(node, "mod") == "builtins.dict"

    def test_set(self):
        node = ast.Set(elts=[])
        assert _infer_assign_type(node, "mod") == "builtins.set"

    def test_tuple(self):
        node = ast.Tuple(elts=[])
        assert _infer_assign_type(node, "mod") == "builtins.tuple"

    def test_unknown_fallback(self):
        node = ast.Name(id="something")
        assert _infer_assign_type(node, "mod") == "builtins.object"


# ── convert_stub_file tests ──────────────────────────────────────────────────


class TestConvertStubFile:
    def test_basic_stub(self, tmp_path: Path):
        stub = tmp_path / "mod.pyi"
        stub.write_text(textwrap.dedent("""
            VERSION: str

            codes: LookupDict

            def get(url: str) -> Response: ...

            class Response:
                status_code: int
                def json(self) -> dict: ...
        """))
        result = convert_stub_file(stub, "requests", "typeshed")
        assert result is not None
        assert result["module"] == "requests"
        assert "get" in result["functions"]
        assert "Response" in result["classes"]
        # VERSION: str is ast.AnnAssign -> goes to attributes, not constants
        assert "VERSION" in result["attributes"]
        assert "codes" in result["attributes"]

    def test_syntax_error_returns_none(self, tmp_path: Path):
        stub = tmp_path / "bad.pyi"
        stub.write_text("def broken(: ...")
        result = convert_stub_file(stub, "mod", "typeshed")
        assert result is None

    def test_overloads_at_module_level(self, tmp_path: Path):
        stub = tmp_path / "mod.pyi"
        stub.write_text(textwrap.dedent("""
            from typing import overload

            @overload
            def loads(s: str) -> dict: ...
            @overload
            def loads(s: bytes) -> dict: ...
        """))
        result = convert_stub_file(stub, "json", "typeshed")
        assert result is not None
        assert "loads" in result["functions"]
        assert result["functions"]["loads"]["return_type"] == "builtins.dict"

    def test_async_function(self, tmp_path: Path):
        stub = tmp_path / "mod.pyi"
        stub.write_text("async def fetch(url: str) -> bytes: ...")
        result = convert_stub_file(stub, "httpx", "typeshed")
        assert result is not None
        assert "fetch" in result["functions"]
        assert result["functions"]["fetch"]["return_type"] == "builtins.bytes"


# ── convert_package tests ────────────────────────────────────────────────────


class TestConvertPackage:
    def test_single_file_package(self, tmp_path: Path):
        pkg = tmp_path / "mypkg"
        inner = pkg / "mypkg"
        inner.mkdir(parents=True)
        (inner / "__init__.pyi").write_text("def hello() -> str: ...")
        result = convert_package(pkg, "mypkg", "typeshed")
        assert result["module"] == "mypkg"
        assert "hello" in result["functions"]

    def test_submodule_merging(self, tmp_path: Path):
        pkg = tmp_path / "flask"
        inner = pkg / "flask"
        inner.mkdir(parents=True)
        (inner / "__init__.pyi").write_text(textwrap.dedent("""
            from .app import Flask as Flask
        """))
        (inner / "app.pyi").write_text(textwrap.dedent("""
            class Flask:
                def run(self) -> None: ...
        """))
        result = convert_package(pkg, "flask", "typeshed")
        # Submodule entry
        assert "app.Flask" in result["classes"]
        # Re-exported to top level
        assert "Flask" in result["classes"]

    def test_empty_package(self, tmp_path: Path):
        pkg = tmp_path / "empty"
        pkg.mkdir()
        result = convert_package(pkg, "empty", "typeshed")
        assert result["module"] == "empty"
        assert result["functions"] == {}
        assert result["classes"] == {}

    def test_pep561_py_files(self, tmp_path: Path):
        pkg = tmp_path / "click"
        pkg.mkdir()
        (pkg / "__init__.py").write_text(textwrap.dedent("""
            from .core import command as command
        """))
        (pkg / "core.py").write_text(textwrap.dedent("""
            def command(name: str) -> None:
                '''Create a new command.'''
                pass
        """))
        result = convert_package(tmp_path, "click", "pep561", ".py")
        assert "command" in result["functions"] or "core.command" in result["functions"]


# ── _find_inner_package_dir tests ─────────────────────────────────────────────


class TestFindInnerPackageDir:
    def test_typeshed_structure(self, tmp_path: Path):
        pkg = tmp_path / "requests"
        inner = pkg / "requests"
        inner.mkdir(parents=True)
        (inner / "__init__.pyi").write_text("")
        assert _find_inner_package_dir(pkg, "requests") == inner

    def test_lowercase_match(self, tmp_path: Path):
        pkg = tmp_path / "PyYAML"
        inner = pkg / "pyyaml"
        inner.mkdir(parents=True)
        (inner / "__init__.pyi").write_text("")
        assert _find_inner_package_dir(pkg, "PyYAML") == inner

    def test_direct_path(self, tmp_path: Path):
        pkg = tmp_path / "mypackage"
        pkg.mkdir()
        (pkg / "core.pyi").write_text("")
        assert _find_inner_package_dir(pkg, "mypackage") == pkg

    def test_not_found(self, tmp_path: Path):
        pkg = tmp_path / "empty"
        pkg.mkdir()
        assert _find_inner_package_dir(pkg, "nonexistent") is None

    def test_init_discovery(self, tmp_path: Path):
        pkg = tmp_path / "PyYAML"
        inner = pkg / "yaml"
        inner.mkdir(parents=True)
        (inner / "__init__.pyi").write_text("")
        result = _find_inner_package_dir(pkg, "PyYAML")
        assert result == inner


# ── _typeshed_importable_name tests ──────────────────────────────────────────


class TestTypeshedImportableName:
    def test_matching_inner_dir(self, tmp_path: Path):
        pkg = tmp_path / "PyYAML"
        inner = pkg / "yaml"
        inner.mkdir(parents=True)
        (inner / "__init__.pyi").write_text("")
        assert _typeshed_importable_name(pkg) == "yaml"

    def test_fallback_lowercase(self, tmp_path: Path):
        pkg = tmp_path / "SomePackage"
        pkg.mkdir()
        (pkg / "mod.pyi").write_text("")
        assert _typeshed_importable_name(pkg) == "somepackage"


# ── _extract_reexports tests ─────────────────────────────────────────────────


class TestExtractReexports:
    def test_explicit_reexport(self, tmp_path: Path):
        init = tmp_path / "__init__.pyi"
        init.write_text(textwrap.dedent("""
            from .api import get as get, post as post
            from .models import Response as Response
        """))
        exports = _extract_reexports(init)
        assert "get" in exports
        assert "post" in exports
        assert "Response" in exports

    def test_implicit_reexport(self, tmp_path: Path):
        init = tmp_path / "__init__.pyi"
        init.write_text("from .core import Flask\n")
        exports = _extract_reexports(init)
        assert "Flask" in exports

    def test_star_import_skipped(self, tmp_path: Path):
        init = tmp_path / "__init__.pyi"
        init.write_text("from .core import *\n")
        exports = _extract_reexports(init)
        assert "*" not in exports

    def test_syntax_error(self, tmp_path: Path):
        init = tmp_path / "__init__.pyi"
        init.write_text("from . import (broken\n")
        exports = _extract_reexports(init)
        assert exports == set()


# ── generate_manifest tests ──────────────────────────────────────────────────


class TestGenerateManifest:
    def test_manifest_structure(self, tmp_path: Path):
        modules = {
            "requests": {
                "module": "requests",
                "python_version": "any",
                "generated_at": "2026-01-01T00:00:00",
                "functions": {"get": {"return_type": "requests.Response", "confidence": 0.95, "params": [], "source": "typeshed"}},
                "classes": {"Response": {"type": "class", "methods": {}, "attributes": {}, "bases": []}},
                "constants": {},
                "attributes": {},
            }
        }
        manifest = generate_manifest(tmp_path, modules, "https://cdn.example.com")
        assert manifest["schema_version"] == "1.0.0"
        assert manifest["statistics"]["total_modules"] == 1
        assert manifest["statistics"]["total_functions"] == 1
        assert manifest["statistics"]["total_classes"] == 1
        assert len(manifest["modules"]) == 1
        entry = manifest["modules"][0]
        assert entry["name"] == "requests"
        assert entry["file"] == "requests_thirdparty.json"
        assert entry["checksum"].startswith("sha256:")
        assert entry["size_bytes"] > 0

        # Verify JSON file was written
        json_file = tmp_path / "requests_thirdparty.json"
        assert json_file.exists()
        data = json.loads(json_file.read_text())
        assert data["module"] == "requests"

        # Verify manifest.json was written
        manifest_file = tmp_path / "manifest.json"
        assert manifest_file.exists()


# ── load_sources tests ────────────────────────────────────────────────────────


class TestLoadSources:
    def test_load_yaml(self, tmp_path: Path):
        config = tmp_path / "sources.yaml"
        config.write_text(textwrap.dedent("""
            sources:
              - type: typeshed
                path: ./typeshed/stubs
              - type: pyi_repo
                path: ./django-stubs
                package: django
        """))
        sources = load_sources(config)
        assert len(sources) == 2
        assert sources[0]["type"] == "typeshed"
        assert sources[1]["package"] == "django"

    def test_empty_config(self, tmp_path: Path):
        config = tmp_path / "sources.yaml"
        config.write_text("sources: []\n")
        sources = load_sources(config)
        assert sources == []


# ── Integration-style tests ──────────────────────────────────────────────────


class TestIntegration:
    """Integration tests using realistic stub content."""

    def test_requests_style_package(self, tmp_path: Path):
        """Simulate converting a requests-like package."""
        pkg = tmp_path / "requests"
        inner = pkg / "requests"
        inner.mkdir(parents=True)

        (inner / "__init__.pyi").write_text(textwrap.dedent("""
            from .api import get as get, post as post
            from .models import Response as Response
            from .sessions import Session as Session
        """))

        (inner / "api.pyi").write_text(textwrap.dedent("""
            from .models import Response

            def get(url: str, **kwargs: object) -> Response: ...
            def post(url: str, data: dict | None = None, **kwargs: object) -> Response: ...
        """))

        (inner / "models.pyi").write_text(textwrap.dedent("""
            class Response:
                status_code: int
                headers: dict
                text: str
                @property
                def content(self) -> bytes: ...
                @property
                def ok(self) -> bool: ...
                def json(self) -> dict: ...
                def raise_for_status(self) -> None: ...
        """))

        (inner / "sessions.pyi").write_text(textwrap.dedent("""
            from .models import Response

            class Session:
                def __init__(self) -> None: ...
                def get(self, url: str, **kwargs: object) -> Response: ...
                def post(self, url: str, data: dict | None = None, **kwargs: object) -> Response: ...
                def close(self) -> None: ...
        """))

        result = convert_package(pkg, "requests", "typeshed")

        # Top-level re-exports
        assert "get" in result["functions"]
        assert "post" in result["functions"]
        assert "Response" in result["classes"]
        assert "Session" in result["classes"]

        # Submodule entries
        assert "api.get" in result["functions"]
        assert "models.Response" in result["classes"]
        assert "sessions.Session" in result["classes"]

        # Response class details
        resp = result["classes"]["Response"]
        assert "json" in resp["methods"]
        assert "raise_for_status" in resp["methods"]
        assert "status_code" in resp["attributes"]
        assert resp["attributes"]["status_code"]["kind"] == "attribute"
        assert "content" in resp["attributes"]
        assert resp["attributes"]["content"]["kind"] == "property"
        assert resp["attributes"]["content"]["type"] == "builtins.bytes"

        # Session __init__ return type
        session = result["classes"]["Session"]
        assert session["methods"]["__init__"]["return_type"] == "requests.sessions.Session"

    def test_django_style_stubs(self, tmp_path: Path):
        """Simulate converting django-stubs."""
        stubs = tmp_path / "django-stubs"
        http_dir = stubs / "http"
        http_dir.mkdir(parents=True)
        views_dir = stubs / "views"
        views_dir.mkdir(parents=True)

        (stubs / "__init__.pyi").write_text("")
        (http_dir / "__init__.pyi").write_text("")
        (http_dir / "request.pyi").write_text(textwrap.dedent("""
            from django.utils.datastructures import QueryDict

            class HttpRequest:
                method: str
                GET: QueryDict
                POST: QueryDict
                path: str
                @property
                def body(self) -> bytes: ...
        """))

        (views_dir / "__init__.pyi").write_text("")
        (views_dir / "generic.pyi").write_text(textwrap.dedent("""
            from django.http.request import HttpRequest

            class View:
                request: HttpRequest
                def dispatch(self, request: HttpRequest, *args: object, **kwargs: object) -> None: ...
                def get(self, request: HttpRequest) -> None: ...
        """))

        result = convert_package(tmp_path, "django", "django-stubs", ".pyi")

        # Check http.request.HttpRequest
        assert "http.request.HttpRequest" in result["classes"]
        http_req = result["classes"]["http.request.HttpRequest"]
        assert "GET" in http_req["attributes"]
        assert "POST" in http_req["attributes"]
        assert "body" in http_req["attributes"]
        assert http_req["attributes"]["body"]["kind"] == "property"

        # Check views.generic.View
        assert "views.generic.View" in result["classes"]
        view = result["classes"]["views.generic.View"]
        assert "get" in view["methods"]
        assert "dispatch" in view["methods"]

    def test_end_to_end_manifest(self, tmp_path: Path):
        """Full pipeline: create stubs, convert, generate manifest."""
        stubs_dir = tmp_path / "stubs"

        # Create a simple package
        pkg = stubs_dir / "simplepkg" / "simplepkg"
        pkg.mkdir(parents=True)
        (pkg / "__init__.pyi").write_text(textwrap.dedent("""
            VERSION: str
            def hello(name: str) -> str: ...
            class Greeter:
                def greet(self, name: str) -> str: ...
        """))

        # Convert
        modules: dict[str, Any] = {}
        mod_data = convert_package(stubs_dir / "simplepkg", "simplepkg", "typeshed")
        modules["simplepkg"] = mod_data

        # Generate manifest
        output_dir = tmp_path / "output"
        output_dir.mkdir()
        manifest = generate_manifest(output_dir, modules, "https://cdn.example.com")

        assert manifest["statistics"]["total_modules"] == 1
        assert manifest["statistics"]["total_functions"] == 1
        assert manifest["statistics"]["total_classes"] == 1
        # VERSION: str is ast.AnnAssign -> attribute, not constant
        assert manifest["statistics"]["total_attributes"] >= 1

        # Verify output files
        assert (output_dir / "manifest.json").exists()
        assert (output_dir / "simplepkg_thirdparty.json").exists()

        # Verify JSON is valid and deserializable
        mod_json = json.loads((output_dir / "simplepkg_thirdparty.json").read_text())
        assert mod_json["module"] == "simplepkg"
        assert "hello" in mod_json["functions"]
        assert "Greeter" in mod_json["classes"]

    def test_typeshed_source_processing(self, tmp_path: Path):
        """Test process_typeshed_source with real directory structure."""
        from convert import process_typeshed_source

        stubs = tmp_path / "stubs"
        pkg = stubs / "mypkg" / "mypkg"
        pkg.mkdir(parents=True)
        (pkg / "__init__.pyi").write_text("def greet(name: str) -> str: ...")

        result = process_typeshed_source({"path": str(stubs)})
        assert "mypkg" in result
        assert "greet" in result["mypkg"]["functions"]

    def test_typeshed_source_missing_path(self, tmp_path: Path):
        """process_typeshed_source with non-existent path returns empty."""
        from convert import process_typeshed_source

        result = process_typeshed_source({"path": str(tmp_path / "nonexistent")})
        assert result == {}

    def test_pyi_repo_source_processing(self, tmp_path: Path):
        """Test process_pyi_repo_source."""
        from convert import process_pyi_repo_source

        stubs = tmp_path / "django-stubs"
        stubs.mkdir()
        (stubs / "__init__.pyi").write_text("")
        (stubs / "core.pyi").write_text("def setup() -> None: ...")

        result = process_pyi_repo_source({
            "path": str(stubs),
            "package": "django",
            "source_name": "django-stubs",
        })
        assert "django" in result

    def test_pyi_repo_source_missing_path(self, tmp_path: Path):
        """process_pyi_repo_source with non-existent path returns empty."""
        from convert import process_pyi_repo_source

        result = process_pyi_repo_source({
            "path": str(tmp_path / "nonexistent"),
            "package": "django",
        })
        assert result == {}

    def test_pep561_source_processing(self, tmp_path: Path):
        """Test process_pep561_source."""
        from convert import process_pep561_source

        pkg = tmp_path / "flask"
        pkg.mkdir()
        (pkg / "__init__.py").write_text("def run() -> None: ...")

        result = process_pep561_source({
            "path": str(pkg),
            "package": "flask",
        })
        assert "flask" in result

    def test_pep561_source_missing_path(self, tmp_path: Path):
        """process_pep561_source with non-existent path returns empty."""
        from convert import process_pep561_source

        result = process_pep561_source({
            "path": str(tmp_path / "nonexistent"),
            "package": "flask",
        })
        assert result == {}

    def test_main_typeshed_mode(self, tmp_path: Path):
        """Test CLI main() with --typeshed flag."""
        from convert import main

        stubs = tmp_path / "stubs"
        pkg = stubs / "testpkg" / "testpkg"
        pkg.mkdir(parents=True)
        (pkg / "__init__.pyi").write_text("def hello() -> str: ...")

        output = tmp_path / "output"
        import sys
        old_argv = sys.argv
        sys.argv = ["convert.py", "--typeshed", str(stubs), "--output", str(output)]
        try:
            ret = main()
        finally:
            sys.argv = old_argv

        assert ret == 0
        assert (output / "manifest.json").exists()

    def test_main_config_mode(self, tmp_path: Path):
        """Test CLI main() with --config flag."""
        from convert import main

        # Create typeshed stubs
        stubs = tmp_path / "stubs"
        pkg = stubs / "testpkg" / "testpkg"
        pkg.mkdir(parents=True)
        (pkg / "__init__.pyi").write_text("def hello() -> str: ...")

        # Create config
        config = tmp_path / "sources.yaml"
        config.write_text(f"sources:\n  - type: typeshed\n    path: {stubs}\n")

        output = tmp_path / "output"
        import sys
        old_argv = sys.argv
        sys.argv = ["convert.py", "--config", str(config), "--output", str(output)]
        try:
            ret = main()
        finally:
            sys.argv = old_argv

        assert ret == 0
        assert (output / "manifest.json").exists()

    def test_main_no_args_fails(self, tmp_path: Path):
        """Test CLI main() without required args fails."""
        from convert import main

        output = tmp_path / "output"
        import sys
        old_argv = sys.argv
        sys.argv = ["convert.py", "--output", str(output)]
        try:
            ret = main()
        finally:
            sys.argv = old_argv

        assert ret == 1

    def test_main_unknown_source_type(self, tmp_path: Path):
        """Test CLI main() with unknown source type in config."""
        from convert import main

        stubs = tmp_path / "stubs"
        pkg = stubs / "testpkg" / "testpkg"
        pkg.mkdir(parents=True)
        (pkg / "__init__.pyi").write_text("def hello() -> str: ...")

        config = tmp_path / "sources.yaml"
        config.write_text(f"sources:\n  - type: typeshed\n    path: {stubs}\n  - type: unknown_type\n    path: /tmp\n")

        output = tmp_path / "output"
        import sys
        old_argv = sys.argv
        sys.argv = ["convert.py", "--config", str(config), "--output", str(output)]
        try:
            ret = main()
        finally:
            sys.argv = old_argv

        assert ret == 0  # Unknown types are warned but don't fail

    def test_has_python_files(self, tmp_path: Path):
        """Test _has_python_files helper."""
        from convert import _has_python_files

        empty = tmp_path / "empty"
        empty.mkdir()
        assert _has_python_files(empty) is False

        (empty / "mod.pyi").write_text("")
        assert _has_python_files(empty) is True

    def test_go_schema_compatibility(self, tmp_path: Path):
        """Verify output matches Go StdlibModule JSON schema."""
        stub = tmp_path / "mod.pyi"
        stub.write_text(textwrap.dedent("""
            VERSION: str = "1.0"
            timeout: float

            def connect(host: str, port: int = 80) -> Connection: ...

            class Connection:
                host: str
                @property
                def connected(self) -> bool: ...
                def close(self) -> None: ...
        """))

        result = convert_stub_file(stub, "mymod", "typeshed")
        assert result is not None

        # Verify top-level schema matches StdlibModule
        assert "module" in result
        assert "python_version" in result
        assert "generated_at" in result
        assert "functions" in result
        assert "classes" in result
        assert "constants" in result
        assert "attributes" in result

        # Verify function schema matches StdlibFunction
        func = result["functions"]["connect"]
        assert "return_type" in func
        assert "confidence" in func
        assert "params" in func
        assert "source" in func

        # Verify param schema matches FunctionParam
        param = func["params"][0]
        assert "name" in param
        assert "type" in param
        assert "required" in param

        # Verify class schema matches StdlibClass
        cls = result["classes"]["Connection"]
        assert "type" in cls
        assert "methods" in cls
        # Extended schema (new fields)
        assert "attributes" in cls
        assert "bases" in cls

        # Verify method schema
        method = cls["methods"]["close"]
        assert "return_type" in method
        assert "confidence" in method
        assert "params" in method

        # Ensure JSON serializable
        json_str = json.dumps(result)
        assert json.loads(json_str) == result
