"""Tests for C3 MRO computation and inheritance flattening."""

from __future__ import annotations

import textwrap
from pathlib import Path
from typing import Any

import pytest

from mro import (
    _c3_merge,
    _fallback_mro,
    _normalize_dep_name,
    build_class_registry,
    build_dependency_graph,
    c3_linearize,
    flatten_inheritance,
    parse_metadata_requires,
    topological_sort,
)


# ── Fixtures ──────────────────────────────────────────────────────────────────


def _make_class(
    bases: list[str] | None = None,
    methods: dict[str, Any] | None = None,
    attributes: dict[str, Any] | None = None,
) -> dict[str, Any]:
    """Helper to build a class data dict."""
    return {
        "type": "class",
        "bases": bases or [],
        "methods": methods or {},
        "attributes": attributes or {},
    }


def _make_method(return_type: str = "builtins.NoneType") -> dict[str, Any]:
    return {
        "return_type": return_type,
        "confidence": 0.95,
        "params": [],
        "source": "typeshed",
    }


def _make_attr(type_str: str = "builtins.str", kind: str = "attribute") -> dict[str, Any]:
    return {
        "type": type_str,
        "confidence": 0.95,
        "source": "typeshed",
        "kind": kind,
    }


# ── parse_metadata_requires tests ────────────────────────────────────────────


class TestParseMetadataRequires:
    def test_basic_requires(self, tmp_path: Path):
        meta = tmp_path / "METADATA.toml"
        meta.write_text('version = "1.0"\nrequires = ["types-requests", "urllib3>=2"]\n')
        result = parse_metadata_requires(meta)
        assert "requests" in result
        assert "urllib3" in result

    def test_no_requires(self, tmp_path: Path):
        meta = tmp_path / "METADATA.toml"
        meta.write_text('version = "1.0"\n')
        result = parse_metadata_requires(meta)
        assert result == []

    def test_missing_file(self, tmp_path: Path):
        meta = tmp_path / "nonexistent.toml"
        result = parse_metadata_requires(meta)
        assert result == []

    def test_complex_requires(self, tmp_path: Path):
        meta = tmp_path / "METADATA.toml"
        meta.write_text('requires = ["types-paramiko", "types-requests", "urllib3>=2"]\n')
        result = parse_metadata_requires(meta)
        assert "paramiko" in result
        assert "requests" in result
        assert "urllib3" in result

    def test_single_quotes(self, tmp_path: Path):
        meta = tmp_path / "METADATA.toml"
        meta.write_text("requires = ['numpy>=1.20', 'pandas-stubs']\n")
        result = parse_metadata_requires(meta)
        assert "numpy" in result
        assert "pandas_stubs" in result


# ── _normalize_dep_name tests ────────────────────────────────────────────────


class TestNormalizeDepName:
    @pytest.mark.parametrize("input_name,expected", [
        ("types-requests", "requests"),
        ("urllib3>=2", "urllib3"),
        ("numpy>=1.20", "numpy"),
        ("cryptography>=37.0.0", "cryptography"),
        ("types-paramiko", "paramiko"),
        ("pandas-stubs", "pandas_stubs"),
        ("types-PyYAML", "pyyaml"),
        ("six", "six"),
        ("mypy~=1.0", "mypy"),
    ])
    def test_normalize(self, input_name: str, expected: str):
        assert _normalize_dep_name(input_name) == expected


# ── topological_sort tests ───────────────────────────────────────────────────


class TestTopologicalSort:
    def test_simple_chain(self):
        graph = {"c": ["b"], "b": ["a"], "a": []}
        result = topological_sort(graph)
        assert result.index("a") < result.index("b")
        assert result.index("b") < result.index("c")

    def test_no_deps(self):
        graph = {"a": [], "b": [], "c": []}
        result = topological_sort(graph)
        assert set(result) == {"a", "b", "c"}

    def test_diamond(self):
        graph = {"d": ["b", "c"], "b": ["a"], "c": ["a"], "a": []}
        result = topological_sort(graph)
        assert result.index("a") < result.index("b")
        assert result.index("a") < result.index("c")
        assert result.index("b") < result.index("d")
        assert result.index("c") < result.index("d")

    def test_circular_dep(self):
        graph = {"a": ["b"], "b": ["a"]}
        result = topological_sort(graph)
        # Should not crash, just break the cycle
        assert len(result) == 2

    def test_empty_graph(self):
        assert topological_sort({}) == []


# ── C3 Linearization tests ──────────────────────────────────────────────────


class TestC3Linearize:
    def test_no_bases(self):
        registry = {"A": _make_class()}
        mro = c3_linearize("A", registry)
        assert mro == ["A", "builtins.object"]

    def test_single_inheritance(self):
        registry = {
            "B": _make_class(bases=["A"]),
            "A": _make_class(),
        }
        mro = c3_linearize("B", registry)
        assert mro == ["B", "A", "builtins.object"]

    def test_linear_chain(self):
        registry = {
            "C": _make_class(bases=["B"]),
            "B": _make_class(bases=["A"]),
            "A": _make_class(),
        }
        mro = c3_linearize("C", registry)
        assert mro == ["C", "B", "A", "builtins.object"]

    def test_multiple_inheritance(self):
        registry = {
            "D": _make_class(bases=["B", "C"]),
            "B": _make_class(bases=["A"]),
            "C": _make_class(bases=["A"]),
            "A": _make_class(),
        }
        mro = c3_linearize("D", registry)
        # Classic diamond: D -> B -> C -> A -> object
        assert mro[0] == "D"
        assert mro[-1] == "builtins.object"
        assert mro.index("B") < mro.index("C")
        assert mro.index("C") < mro.index("A")

    def test_unknown_class(self):
        registry: dict[str, dict[str, Any]] = {}
        mro = c3_linearize("Unknown", registry)
        assert mro == ["Unknown"]

    def test_deep_chain(self):
        registry = {
            "E": _make_class(bases=["D"]),
            "D": _make_class(bases=["C"]),
            "C": _make_class(bases=["B"]),
            "B": _make_class(bases=["A"]),
            "A": _make_class(),
        }
        mro = c3_linearize("E", registry)
        assert mro == ["E", "D", "C", "B", "A", "builtins.object"]

    def test_missing_base_class(self):
        """Base class not in registry — should still produce valid MRO."""
        registry = {
            "Child": _make_class(bases=["MissingBase"]),
        }
        mro = c3_linearize("Child", registry)
        assert mro[0] == "Child"
        assert "MissingBase" in mro
        assert mro[-1] == "builtins.object"

    def test_object_base_deduped(self):
        """builtins.object should appear only once, at the end."""
        registry = {
            "B": _make_class(bases=["A"]),
            "A": _make_class(bases=["builtins.object"]),
            "builtins.object": _make_class(),
        }
        mro = c3_linearize("B", registry)
        assert mro.count("builtins.object") == 1
        assert mro[-1] == "builtins.object"

    def test_cross_package_inheritance(self):
        """Simulate docker.APIClient inheriting requests.Session."""
        registry = {
            "docker.APIClient": _make_class(bases=["requests.Session", "docker.BuildApiMixin"]),
            "requests.Session": _make_class(
                bases=["requests.SessionRedirectMixin"],
                methods={"get": _make_method("requests.Response")},
            ),
            "requests.SessionRedirectMixin": _make_class(),
            "docker.BuildApiMixin": _make_class(
                methods={"build": _make_method("builtins.dict")},
            ),
        }
        mro = c3_linearize("docker.APIClient", registry)
        assert mro[0] == "docker.APIClient"
        assert "requests.Session" in mro
        assert "docker.BuildApiMixin" in mro
        assert mro[-1] == "builtins.object"


class TestC3Merge:
    def test_empty(self):
        assert _c3_merge([]) == []

    def test_single(self):
        assert _c3_merge([["A", "B"]]) == ["A", "B"]

    def test_inconsistent_returns_none(self):
        # Impossible hierarchy: A before B AND B before A
        result = _c3_merge([["A", "B"], ["B", "A"]])
        assert result is None


class TestFallbackMro:
    def test_simple(self):
        registry = {
            "C": _make_class(bases=["B", "A"]),
            "B": _make_class(),
            "A": _make_class(),
        }
        result = _fallback_mro("C", registry)
        assert result[0] == "C"
        assert "B" in result
        assert "A" in result

    def test_skips_object(self):
        registry = {
            "A": _make_class(bases=["builtins.object"]),
        }
        result = _fallback_mro("A", registry)
        assert "builtins.object" not in result


# ── build_class_registry tests ──────────────────────────────────────────────


class TestBuildClassRegistry:
    def test_basic(self):
        modules = {
            "requests": {
                "classes": {
                    "Response": _make_class(),
                    "Session": _make_class(),
                }
            },
            "docker": {
                "classes": {
                    "APIClient": _make_class(bases=["requests.Session"]),
                }
            },
        }
        registry = build_class_registry(modules)
        assert "requests.Response" in registry
        assert "requests.Session" in registry
        assert "docker.APIClient" in registry

    def test_empty_modules(self):
        registry = build_class_registry({})
        assert registry == {}

    def test_module_without_classes(self):
        modules = {"mod": {"functions": {"f": {}}}}
        registry = build_class_registry(modules)
        assert registry == {}


# ── flatten_inheritance tests ────────────────────────────────────────────────


class TestFlattenInheritance:
    def test_single_inheritance_flattening(self):
        modules = {
            "mod": {
                "classes": {
                    "Parent": _make_class(
                        methods={"greet": _make_method("builtins.str")},
                        attributes={"name": _make_attr()},
                    ),
                    "Child": _make_class(
                        bases=["mod.Parent"],
                        methods={"play": _make_method()},
                    ),
                }
            }
        }
        flatten_inheritance(modules)

        child = modules["mod"]["classes"]["Child"]
        assert "mro" in child
        assert child["mro"][0] == "mod.Child"
        assert "mod.Parent" in child["mro"]

        # Inherited method
        assert "inherited_methods" in child
        assert "greet" in child["inherited_methods"]
        assert child["inherited_methods"]["greet"]["inherited_from"] == "mod.Parent"

        # Inherited attribute
        assert "inherited_attributes" in child
        assert "name" in child["inherited_attributes"]

    def test_overridden_method_not_inherited(self):
        modules = {
            "mod": {
                "classes": {
                    "Parent": _make_class(
                        methods={"save": _make_method()},
                    ),
                    "Child": _make_class(
                        bases=["mod.Parent"],
                        methods={"save": _make_method("builtins.bool")},
                    ),
                }
            }
        }
        flatten_inheritance(modules)

        child = modules["mod"]["classes"]["Child"]
        # save is overridden, should NOT be in inherited_methods
        inherited = child.get("inherited_methods", {})
        assert "save" not in inherited

    def test_diamond_inheritance(self):
        modules = {
            "mod": {
                "classes": {
                    "A": _make_class(
                        methods={"base_method": _make_method()},
                        attributes={"x": _make_attr("builtins.int")},
                    ),
                    "B": _make_class(
                        bases=["mod.A"],
                        methods={"b_method": _make_method()},
                    ),
                    "C": _make_class(
                        bases=["mod.A"],
                        methods={"c_method": _make_method()},
                    ),
                    "D": _make_class(
                        bases=["mod.B", "mod.C"],
                    ),
                }
            }
        }
        flatten_inheritance(modules)

        d = modules["mod"]["classes"]["D"]
        inherited = d.get("inherited_methods", {})
        assert "base_method" in inherited
        assert "b_method" in inherited
        assert "c_method" in inherited

        # x attribute should be inherited
        inherited_attrs = d.get("inherited_attributes", {})
        assert "x" in inherited_attrs

    def test_cross_package_flattening(self):
        modules = {
            "requests": {
                "classes": {
                    "Session": _make_class(
                        methods={
                            "get": _make_method("requests.Response"),
                            "post": _make_method("requests.Response"),
                        },
                    ),
                }
            },
            "docker": {
                "classes": {
                    "APIClient": _make_class(
                        bases=["requests.Session"],
                        methods={"build": _make_method("builtins.dict")},
                    ),
                }
            },
        }
        flatten_inheritance(modules)

        api_client = modules["docker"]["classes"]["APIClient"]
        inherited = api_client.get("inherited_methods", {})
        assert "get" in inherited
        assert "post" in inherited
        assert inherited["get"]["inherited_from"] == "requests.Session"
        assert inherited["get"]["return_type"] == "requests.Response"

        # Own method should NOT be in inherited
        assert "build" not in inherited

    def test_no_inheritance(self):
        modules = {
            "mod": {
                "classes": {
                    "Simple": _make_class(
                        methods={"do_thing": _make_method()},
                    ),
                }
            }
        }
        flatten_inheritance(modules)

        simple = modules["mod"]["classes"]["Simple"]
        assert "mro" in simple
        # No inherited_methods if nothing to inherit
        assert "inherited_methods" not in simple or simple.get("inherited_methods") == {}

    def test_with_prebuilt_registry(self):
        modules = {
            "mod": {
                "classes": {
                    "A": _make_class(methods={"hello": _make_method()}),
                    "B": _make_class(bases=["mod.A"]),
                }
            }
        }
        registry = build_class_registry(modules)
        flatten_inheritance(modules, class_registry=registry)

        b = modules["mod"]["classes"]["B"]
        assert "hello" in b.get("inherited_methods", {})


# ── build_dependency_graph tests ─────────────────────────────────────────────


class TestBuildDependencyGraph:
    def test_basic_graph(self, tmp_path: Path):
        stubs = tmp_path / "stubs"
        # Create docker package with dependency on requests
        docker_dir = stubs / "docker" / "docker"
        docker_dir.mkdir(parents=True)
        (docker_dir / "__init__.pyi").write_text("")
        (stubs / "docker" / "METADATA.toml").write_text(
            'requires = ["types-requests"]\n'
        )

        # Create requests package
        requests_dir = stubs / "requests" / "requests"
        requests_dir.mkdir(parents=True)
        (requests_dir / "__init__.pyi").write_text("")
        (stubs / "requests" / "METADATA.toml").write_text('version = "1.0"\n')

        graph = build_dependency_graph(stubs, ["docker", "requests"])
        assert "requests" in graph["docker"]
        assert graph["requests"] == []

    def test_no_metadata(self, tmp_path: Path):
        stubs = tmp_path / "stubs"
        pkg_dir = stubs / "simple" / "simple"
        pkg_dir.mkdir(parents=True)
        (pkg_dir / "__init__.pyi").write_text("")

        graph = build_dependency_graph(stubs, ["simple"])
        assert graph["simple"] == []


# ── Integration with real typeshed (if available) ────────────────────────────


class TestRealTypeshed:
    """Integration tests against real typeshed stubs."""

    TYPESHED_PATH = Path("/tmp/typeshed/stubs")

    @pytest.fixture(autouse=True)
    def skip_if_no_typeshed(self):
        if not self.TYPESHED_PATH.exists():
            pytest.skip("typeshed not available at /tmp/typeshed/stubs")

    def test_requests_session_mro(self):
        """Verify requests.Session MRO is computed correctly."""
        from convert import convert_package

        pkg_data = convert_package(
            self.TYPESHED_PATH / "requests", "requests", "typeshed"
        )
        modules = {"requests": pkg_data}
        flatten_inheritance(modules)

        # Find Session class
        session = None
        for cls_name, cls_data in pkg_data["classes"].items():
            if cls_name.endswith("Session") or cls_name == "Session":
                session = cls_data
                break

        assert session is not None, "Session class not found"
        assert "mro" in session
        assert session["mro"][0].endswith("Session")

    def test_docker_cross_package_inheritance(self):
        """Test docker.APIClient inherits requests.Session methods."""
        from convert import convert_package

        # Need both packages
        requests_data = convert_package(
            self.TYPESHED_PATH / "requests", "requests", "typeshed"
        )
        docker_data = convert_package(
            self.TYPESHED_PATH / "docker", "docker", "typeshed"
        )
        modules = {"requests": requests_data, "docker": docker_data}
        flatten_inheritance(modules)

        # Find APIClient
        api_client = None
        for cls_name, cls_data in docker_data["classes"].items():
            if "APIClient" in cls_name:
                api_client = cls_data
                break

        assert api_client is not None, "APIClient class not found"
        assert "mro" in api_client

        # Should inherit get/post from requests.Session
        inherited = api_client.get("inherited_methods", {})
        assert "get" in inherited or any("get" in k for k in inherited)

    def test_full_typeshed_mro_no_errors(self):
        """Run MRO on all typeshed packages without errors."""
        from convert import process_typeshed_source

        modules = process_typeshed_source({"path": str(self.TYPESHED_PATH)})
        # Should not raise
        flatten_inheritance(modules)

        # Verify at least some classes got MROs
        total_with_mro = 0
        total_with_inherited = 0
        for mod_data in modules.values():
            for cls_data in mod_data.get("classes", {}).values():
                if "mro" in cls_data:
                    total_with_mro += 1
                if cls_data.get("inherited_methods") or cls_data.get("inherited_attributes"):
                    total_with_inherited += 1

        assert total_with_mro > 100, f"Only {total_with_mro} classes got MRO"
        assert total_with_inherited > 50, f"Only {total_with_inherited} classes got inherited members"
