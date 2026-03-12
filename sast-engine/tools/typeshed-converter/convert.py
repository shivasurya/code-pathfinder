#!/usr/bin/env python3
"""
Typeshed stub-to-JSON converter for Code Pathfinder.

Converts type stubs (.pyi) and PEP 561 annotated source (.py) files
into the StdlibModule JSON format consumed by the Go sast-engine.

Supports three source types:
  - typeshed: Auto-discovers packages under stubs/ directory
  - pyi_repo: External .pyi stub repos (e.g., django-stubs)
  - pep561:   Python packages with inline type annotations

Usage:
    python convert.py --config sources.yaml --output ./output/thirdparty/v1/
    python convert.py --typeshed ./typeshed/stubs --output ./output/thirdparty/v1/
"""

from __future__ import annotations

import argparse
import ast
import hashlib
import json
import logging
import sys
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

import yaml

logger = logging.getLogger(__name__)

# ── Type Annotation Resolution ───────────────────────────────────────────────

BUILTIN_MAP: dict[str, str] = {
    "str": "builtins.str",
    "int": "builtins.int",
    "float": "builtins.float",
    "bool": "builtins.bool",
    "bytes": "builtins.bytes",
    "bytearray": "builtins.bytearray",
    "list": "builtins.list",
    "dict": "builtins.dict",
    "set": "builtins.set",
    "frozenset": "builtins.frozenset",
    "tuple": "builtins.tuple",
    "None": "builtins.NoneType",
    "type": "builtins.type",
    "object": "builtins.object",
    "complex": "builtins.complex",
    "memoryview": "builtins.memoryview",
    "range": "builtins.range",
    "slice": "builtins.slice",
    "super": "builtins.super",
    "Exception": "builtins.Exception",
    "BaseException": "builtins.BaseException",
    "Ellipsis": "builtins.Ellipsis",
}

# Names that should NOT be module-qualified (typing constructs, etc.)
TYPING_NAMES: set[str] = {
    "Any", "Union", "Optional", "List", "Dict", "Set", "Tuple",
    "FrozenSet", "Type", "Callable", "Iterator", "Iterable",
    "Generator", "Coroutine", "AsyncIterator", "AsyncIterable",
    "AsyncGenerator", "Sequence", "Mapping", "MutableMapping",
    "MutableSequence", "MutableSet", "IO", "TextIO", "BinaryIO",
    "Pattern", "Match", "ClassVar", "Final", "Literal",
    "TypeVar", "ParamSpec", "Protocol", "TypeAlias", "Self",
    "Never", "NoReturn", "Concatenate", "Unpack", "TypeVarTuple",
    "TypeGuard", "SupportsInt", "SupportsFloat", "SupportsComplex",
    "SupportsBytes", "SupportsAbs", "SupportsRound", "Reversible",
    "ContextManager", "AsyncContextManager", "Awaitable",
    "Hashable", "Sized", "Container", "Collection", "Buffer",
    "ReadableBuffer", "WriteableBuffer",
}

# Names that map to builtins.object as a safe fallback
OPAQUE_NAMES: set[str] = {
    "Incomplete", "_typeshed.Incomplete",
    "TypeVar", "ParamSpec", "TypeVarTuple",
}


def resolve_type_annotation(annotation: ast.expr | None, module_name: str) -> str:
    """Map a type annotation AST node to a fully qualified type string."""
    if annotation is None:
        return "builtins.NoneType"

    if isinstance(annotation, ast.Constant):
        if annotation.value is None:
            return "builtins.NoneType"
        if isinstance(annotation.value, str):
            # String annotations like "Response" — treat as forward ref
            if annotation.value in BUILTIN_MAP:
                return BUILTIN_MAP[annotation.value]
            if annotation.value in OPAQUE_NAMES:
                return "builtins.object"
            return f"{module_name}.{annotation.value}"
        return f"builtins.{type(annotation.value).__name__}"

    if isinstance(annotation, ast.Name):
        name = annotation.id
        if name in BUILTIN_MAP:
            return BUILTIN_MAP[name]
        if name in OPAQUE_NAMES:
            return "builtins.object"
        if name in TYPING_NAMES:
            return f"typing.{name}"
        return f"{module_name}.{name}"

    if isinstance(annotation, ast.Attribute):
        return _resolve_attribute_fqn(annotation)

    if isinstance(annotation, ast.Subscript):
        base = resolve_type_annotation(annotation.value, module_name)
        # Unwrap Optional[X] -> X
        if base in ("typing.Optional",):
            return resolve_type_annotation(annotation.slice, module_name)
        # Unwrap Union[X, Y] -> first non-None
        if base in ("typing.Union",):
            return _resolve_union(annotation.slice, module_name)
        # Generic types: list[str] -> builtins.list
        return base

    if isinstance(annotation, ast.BinOp) and isinstance(annotation.op, ast.BitOr):
        # PEP 604: X | Y -> first non-None
        left = resolve_type_annotation(annotation.left, module_name)
        if left != "builtins.NoneType":
            return left
        return resolve_type_annotation(annotation.right, module_name)

    if isinstance(annotation, ast.List | ast.Tuple):
        # [str, int] or (str, int) in type context — shouldn't happen often
        return "builtins.object"

    return "builtins.object"


def _resolve_attribute_fqn(node: ast.Attribute) -> str:
    """Resolve ast.Attribute chain to dotted FQN string."""
    parts: list[str] = []
    current: ast.expr = node
    while isinstance(current, ast.Attribute):
        parts.append(current.attr)
        current = current.value
    if isinstance(current, ast.Name):
        parts.append(current.id)
    parts.reverse()
    fqn = ".".join(parts)
    # Check if the root is a builtin
    if parts and parts[0] in BUILTIN_MAP:
        return fqn
    return fqn


def _resolve_union(slice_node: ast.expr, module_name: str) -> str:
    """Resolve Union[X, Y, ...] -> first non-None type."""
    elements: list[ast.expr] = []
    if isinstance(slice_node, ast.Tuple):
        elements = list(slice_node.elts)
    else:
        elements = [slice_node]

    for elem in elements:
        resolved = resolve_type_annotation(elem, module_name)
        if resolved != "builtins.NoneType":
            return resolved
    return "builtins.NoneType"


# ── Extraction Functions ─────────────────────────────────────────────────────


def extract_params(args: ast.arguments, module_name: str) -> list[dict[str, Any]]:
    """Extract function parameters with types."""
    params: list[dict[str, Any]] = []
    # Defaults alignment: defaults apply to last N args
    num_defaults = len(args.defaults)
    num_args = len(args.args)

    for i, arg in enumerate(args.args):
        if arg.arg in ("self", "cls"):
            continue
        param_type = resolve_type_annotation(arg.annotation, module_name)
        has_default = i >= (num_args - num_defaults)
        params.append({
            "name": arg.arg,
            "type": param_type,
            "required": not has_default,
        })

    # *args
    if args.vararg:
        params.append({
            "name": f"*{args.vararg.arg}",
            "type": resolve_type_annotation(args.vararg.annotation, module_name),
            "required": False,
        })

    # keyword-only args
    kw_defaults = args.kw_defaults
    for i, arg in enumerate(args.kwonlyargs):
        has_default = kw_defaults[i] is not None
        params.append({
            "name": arg.arg,
            "type": resolve_type_annotation(arg.annotation, module_name),
            "required": not has_default,
        })

    # **kwargs
    if args.kwarg:
        params.append({
            "name": f"**{args.kwarg.arg}",
            "type": resolve_type_annotation(args.kwarg.annotation, module_name),
            "required": False,
        })

    return params


def extract_function(
    node: ast.FunctionDef | ast.AsyncFunctionDef,
    module_name: str,
    source_type: str,
) -> dict[str, Any]:
    """Extract function signature."""
    return {
        "return_type": resolve_type_annotation(node.returns, module_name),
        "confidence": 0.95,
        "params": extract_params(node.args, module_name),
        "source": source_type,
    }


def extract_class(
    node: ast.ClassDef,
    module_name: str,
    source_type: str,
) -> dict[str, Any]:
    """Extract class with methods, attributes, properties, and base classes."""
    methods: dict[str, Any] = {}
    attributes: dict[str, Any] = {}
    overloads: dict[str, list[ast.FunctionDef]] = {}

    for item in node.body:
        if isinstance(item, ast.FunctionDef | ast.AsyncFunctionDef):
            # Check decorators
            is_property = _has_decorator(item, "property")
            is_overload = _has_decorator(item, "overload")

            if is_property:
                attributes[item.name] = {
                    "type": resolve_type_annotation(item.returns, module_name),
                    "confidence": 0.95,
                    "source": source_type,
                    "kind": "property",
                }
            elif is_overload:
                overloads.setdefault(item.name, []).append(item)
            else:
                func_data = extract_function(item, module_name, source_type)
                # __init__ special case
                if item.name == "__init__":
                    func_data["return_type"] = f"{module_name}.{node.name}"
                methods[item.name] = func_data

        elif isinstance(item, ast.AnnAssign) and isinstance(item.target, ast.Name):
            attributes[item.target.id] = {
                "type": resolve_type_annotation(item.annotation, module_name),
                "confidence": 0.95,
                "source": source_type,
                "kind": "attribute",
            }

    # Resolve overloads: if method not yet in methods (no implementation body),
    # pick first non-None return type from overloads
    for name, sigs in overloads.items():
        if name not in methods:
            resolved_return = "builtins.object"
            best_params: list[dict[str, Any]] = []
            for sig in sigs:
                ret = resolve_type_annotation(sig.returns, module_name)
                if ret != "builtins.NoneType" and resolved_return == "builtins.object":
                    resolved_return = ret
                    best_params = extract_params(sig.args, module_name)
            methods[name] = {
                "return_type": resolved_return,
                "confidence": 0.95,
                "params": best_params if best_params else extract_params(sigs[0].args, module_name),
                "source": source_type,
            }

    # Extract base classes
    bases: list[str] = []
    for base in node.bases:
        base_fqn = resolve_type_annotation(base, module_name)
        if base_fqn not in ("builtins.object",):
            bases.append(base_fqn)

    result: dict[str, Any] = {
        "type": "class",
        "methods": methods,
        "attributes": attributes,
        "bases": bases,
    }
    return result


def _has_decorator(node: ast.FunctionDef | ast.AsyncFunctionDef, name: str) -> bool:
    """Check if a function has a specific decorator."""
    for dec in node.decorator_list:
        if isinstance(dec, ast.Name) and dec.id == name:
            return True
        if isinstance(dec, ast.Attribute) and dec.attr == name:
            return True
    return False


# ── Module Conversion ────────────────────────────────────────────────────────


def convert_stub_file(
    source_path: Path,
    module_name: str,
    source_type: str,
) -> dict[str, Any] | None:
    """Parse a .pyi or .py file and extract type information."""
    try:
        source_text = source_path.read_text(encoding="utf-8", errors="replace")
        tree = ast.parse(source_text, filename=str(source_path))
    except SyntaxError:
        logger.warning("Failed to parse %s: SyntaxError", source_path)
        return None

    functions: dict[str, Any] = {}
    classes: dict[str, Any] = {}
    constants: dict[str, Any] = {}
    attributes: dict[str, Any] = {}
    overloads: dict[str, list[ast.FunctionDef]] = {}

    for node in ast.iter_child_nodes(tree):
        if isinstance(node, ast.FunctionDef | ast.AsyncFunctionDef):
            if _has_decorator(node, "overload"):
                overloads.setdefault(node.name, []).append(node)
            else:
                functions[node.name] = extract_function(node, module_name, source_type)

        elif isinstance(node, ast.ClassDef):
            classes[node.name] = extract_class(node, module_name, source_type)

        elif isinstance(node, ast.AnnAssign) and isinstance(node.target, ast.Name):
            attributes[node.target.id] = {
                "type": resolve_type_annotation(node.annotation, module_name),
                "confidence": 0.95,
                "source": source_type,
            }

        elif isinstance(node, ast.Assign):
            for target in node.targets:
                if isinstance(target, ast.Name):
                    # Try to infer type from value
                    val_type = _infer_assign_type(node.value, module_name)
                    constants[target.id] = {
                        "type": val_type,
                        "value": "",
                        "confidence": 0.95,
                    }

    # Resolve module-level overloads
    for name, sigs in overloads.items():
        if name not in functions:
            resolved_return = "builtins.object"
            best_params: list[dict[str, Any]] = []
            for sig in sigs:
                ret = resolve_type_annotation(sig.returns, module_name)
                if ret != "builtins.NoneType" and resolved_return == "builtins.object":
                    resolved_return = ret
                    best_params = extract_params(sig.args, module_name)
            functions[name] = {
                "return_type": resolved_return,
                "confidence": 0.95,
                "params": best_params if best_params else extract_params(sigs[0].args, module_name),
                "source": source_type,
            }

    return {
        "module": module_name,
        "python_version": "any",
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "functions": functions,
        "classes": classes,
        "constants": constants,
        "attributes": attributes,
    }


def _infer_assign_type(value: ast.expr | None, module_name: str) -> str:
    """Infer type from an assignment value (best effort)."""
    if value is None:
        return "builtins.object"
    if isinstance(value, ast.Constant):
        if value.value is None:
            return "builtins.NoneType"
        return f"builtins.{type(value.value).__name__}"
    if isinstance(value, ast.List):
        return "builtins.list"
    if isinstance(value, ast.Dict):
        return "builtins.dict"
    if isinstance(value, ast.Set):
        return "builtins.set"
    if isinstance(value, ast.Tuple):
        return "builtins.tuple"
    return "builtins.object"


# ── Package Conversion ───────────────────────────────────────────────────────


def convert_package(
    package_path: Path,
    package_name: str,
    source_type: str,
    file_suffix: str = ".pyi",
) -> dict[str, Any]:
    """Convert all stub files in a package directory to a single module dict.

    Handles submodule merging: flask/app.pyi -> classes keyed as "app.ClassName".
    Re-exports from __init__.pyi are promoted to top-level.
    """
    merged: dict[str, Any] = {
        "module": package_name,
        "python_version": "any",
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "functions": {},
        "classes": {},
        "constants": {},
        "attributes": {},
    }

    # Find the inner package directory (e.g., stubs/requests/requests/)
    inner_dir = _find_inner_package_dir(package_path, package_name)
    if inner_dir is None:
        logger.warning("No inner package directory found for %s in %s", package_name, package_path)
        return merged

    # Collect all stub files
    stub_files = sorted(inner_dir.rglob(f"*{file_suffix}"))
    if not stub_files:
        # Try .py as well for PEP 561
        if file_suffix == ".pyi":
            stub_files = sorted(inner_dir.rglob("*.py"))
        if not stub_files:
            logger.warning("No %s files found in %s", file_suffix, inner_dir)
            return merged

    # Parse __init__ first to identify re-exports
    init_file = inner_dir / f"__init__{file_suffix}"
    if not init_file.exists() and file_suffix == ".pyi":
        init_file = inner_dir / "__init__.py"
    re_exports: set[str] = set()
    if init_file.exists():
        re_exports = _extract_reexports(init_file)

    for stub_file in stub_files:
        # Compute submodule name relative to inner_dir
        rel = stub_file.relative_to(inner_dir)
        parts = list(rel.parts)
        # Remove file extension from last part
        parts[-1] = parts[-1].rsplit(".", 1)[0]
        # Remove __init__ — it represents the package itself
        if parts[-1] == "__init__":
            parts = parts[:-1]

        submodule = ".".join(parts) if parts else ""
        full_module = f"{package_name}.{submodule}" if submodule else package_name

        file_data = convert_stub_file(stub_file, full_module, source_type)
        if file_data is None:
            continue

        # Merge into package
        prefix = f"{submodule}." if submodule else ""

        for name, func in file_data.get("functions", {}).items():
            key = f"{prefix}{name}" if submodule else name
            merged["functions"][key] = func
            # Promote re-exports to top-level
            if name in re_exports and submodule:
                merged["functions"][name] = func

        for name, cls in file_data.get("classes", {}).items():
            key = f"{prefix}{name}" if submodule else name
            merged["classes"][key] = cls
            if name in re_exports and submodule:
                merged["classes"][name] = cls

        for name, const in file_data.get("constants", {}).items():
            key = f"{prefix}{name}" if submodule else name
            merged["constants"][key] = const

        for name, attr in file_data.get("attributes", {}).items():
            key = f"{prefix}{name}" if submodule else name
            merged["attributes"][key] = attr

    return merged


def _find_inner_package_dir(package_path: Path, package_name: str) -> Path | None:
    """Find the inner package directory.

    Typeshed structure: stubs/requests/requests/ — inner dir matches package name.
    PEP 561 / pyi_repo: path IS the inner dir already.
    """
    # Normalize package name for filesystem (e.g., PyYAML -> yaml)
    # Check if there's a subdirectory matching the package name
    candidates = [
        package_path / package_name,
        package_path / package_name.lower(),
        package_path / package_name.replace("-", "_"),
        package_path / package_name.replace("-", "_").lower(),
        package_path,  # Direct (for pyi_repo / pep561 sources)
    ]
    for candidate in candidates:
        if candidate.is_dir() and _has_python_files(candidate):
            return candidate

    # For typeshed, look for any directory with __init__.pyi
    for child in package_path.iterdir():
        if child.is_dir() and (child / "__init__.pyi").exists():
            return child

    return None


def _has_python_files(directory: Path) -> bool:
    """Check if directory has .pyi or .py files."""
    return any(directory.glob("*.pyi")) or any(directory.glob("*.py"))


def _extract_reexports(init_file: Path) -> set[str]:
    """Extract re-exported names from __init__.pyi/__init__.py.

    Looks for patterns like:
        from .api import get as get, post as post
        from .models import Response as Response
    """
    try:
        tree = ast.parse(init_file.read_text(encoding="utf-8", errors="replace"))
    except SyntaxError:
        return set()

    exports: set[str] = set()
    for node in ast.iter_child_nodes(tree):
        if isinstance(node, ast.ImportFrom):
            for alias in node.names:
                # "from .x import Y as Y" is the re-export pattern
                # But also "from .x import Y" (implicit re-export)
                name = alias.asname if alias.asname else alias.name
                if name != "*":
                    exports.add(name)
    return exports


# ── Manifest Generation ──────────────────────────────────────────────────────


def generate_manifest(
    output_dir: Path,
    modules: dict[str, dict[str, Any]],
    base_url: str,
) -> dict[str, Any]:
    """Generate manifest.json with checksums and statistics."""
    entries: list[dict[str, Any]] = []
    total_functions = 0
    total_classes = 0
    total_constants = 0
    total_attributes = 0

    for mod_name, mod_data in sorted(modules.items()):
        filename = f"{mod_name.replace('.', '_')}_thirdparty.json"
        filepath = output_dir / filename
        content = json.dumps(mod_data, indent=2, sort_keys=True)
        filepath.write_text(content, encoding="utf-8")

        checksum = hashlib.sha256(content.encode("utf-8")).hexdigest()
        size = filepath.stat().st_size

        entries.append({
            "name": mod_name,
            "file": filename,
            "url": f"{base_url}/{filename}",
            "size_bytes": size,
            "checksum": f"sha256:{checksum}",
        })

        total_functions += len(mod_data.get("functions", {}))
        total_classes += len(mod_data.get("classes", {}))
        total_constants += len(mod_data.get("constants", {}))
        total_attributes += len(mod_data.get("attributes", {}))

    manifest = {
        "schema_version": "1.0.0",
        "registry_version": "v1",
        "python_version": {
            "major": 0,
            "minor": 0,
            "patch": 0,
            "full": "any",
        },
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "generator_version": "1.0.0",
        "base_url": base_url,
        "modules": entries,
        "statistics": {
            "total_modules": len(entries),
            "total_functions": total_functions,
            "total_classes": total_classes,
            "total_constants": total_constants,
            "total_attributes": total_attributes,
        },
    }

    manifest_path = output_dir / "manifest.json"
    manifest_path.write_text(
        json.dumps(manifest, indent=2, sort_keys=True), encoding="utf-8"
    )
    return manifest


# ── Source Loading ────────────────────────────────────────────────────────────


def load_sources(config_path: Path) -> list[dict[str, Any]]:
    """Load source configuration from YAML."""
    with open(config_path, encoding="utf-8") as f:
        config = yaml.safe_load(f)
    return config.get("sources", [])


def process_typeshed_source(source: dict[str, Any]) -> dict[str, dict[str, Any]]:
    """Process typeshed stubs/ directory — auto-discover all packages."""
    stubs_path = Path(source["path"])
    if not stubs_path.exists():
        logger.error("Typeshed stubs path not found: %s", stubs_path)
        return {}

    modules: dict[str, dict[str, Any]] = {}
    for pkg_dir in sorted(stubs_path.iterdir()):
        if not pkg_dir.is_dir() or pkg_dir.name.startswith("."):
            continue
        # Package name is the directory name (e.g., "requests", "PyYAML")
        # But importable name may differ — check for inner directory
        pkg_name = _typeshed_importable_name(pkg_dir)
        logger.info("Converting typeshed package: %s -> %s", pkg_dir.name, pkg_name)

        mod_data = convert_package(pkg_dir, pkg_name, "typeshed")
        if mod_data["functions"] or mod_data["classes"] or mod_data["attributes"]:
            modules[pkg_name] = mod_data

    return modules


def _typeshed_importable_name(pkg_dir: Path) -> str:
    """Determine the importable module name for a typeshed package.

    Typeshed directory name is PyPI name (e.g., "PyYAML"),
    but the importable name may differ (e.g., "yaml").
    """
    # Check for inner directories that have __init__.pyi
    for child in pkg_dir.iterdir():
        if child.is_dir() and (child / "__init__.pyi").exists():
            return child.name

    # Fallback to lowercase directory name
    return pkg_dir.name.lower().replace("-", "_")


def process_pyi_repo_source(source: dict[str, Any]) -> dict[str, dict[str, Any]]:
    """Process an external .pyi stub repo (e.g., django-stubs)."""
    path = Path(source["path"])
    package = source["package"]
    if not path.exists():
        logger.error("pyi_repo path not found: %s", path)
        return {}

    logger.info("Converting pyi_repo: %s -> %s", path, package)
    mod_data = convert_package(path, package, source.get("source_name", "external-stubs"), ".pyi")
    if mod_data["functions"] or mod_data["classes"] or mod_data["attributes"]:
        return {package: mod_data}
    return {}


def process_pep561_source(source: dict[str, Any]) -> dict[str, dict[str, Any]]:
    """Process a PEP 561 package with inline annotations."""
    path = Path(source["path"])
    package = source["package"]
    if not path.exists():
        logger.error("pep561 path not found: %s", path)
        return {}

    logger.info("Converting PEP 561 package: %s -> %s", path, package)
    mod_data = convert_package(path, package, "pep561", ".py")
    if mod_data["functions"] or mod_data["classes"] or mod_data["attributes"]:
        return {package: mod_data}
    return {}


# ── CLI ───────────────────────────────────────────────────────────────────────


def main() -> int:
    parser = argparse.ArgumentParser(description="Convert type stubs to registry JSON")
    parser.add_argument("--config", type=Path, help="Path to sources.yaml config")
    parser.add_argument("--typeshed", type=Path, help="Path to typeshed stubs/ directory (standalone mode)")
    parser.add_argument("--output", type=Path, required=True, help="Output directory for JSON files")
    parser.add_argument(
        "--base-url",
        default="https://assets.codepathfinder.dev/registries/thirdparty/v1",
        help="CDN base URL for manifest",
    )
    parser.add_argument("--no-mro", action="store_true", help="Skip MRO computation and inheritance flattening")
    parser.add_argument("--verbose", "-v", action="store_true")
    args = parser.parse_args()

    logging.basicConfig(
        level=logging.DEBUG if args.verbose else logging.INFO,
        format="%(levelname)s: %(message)s",
    )

    args.output.mkdir(parents=True, exist_ok=True)

    all_modules: dict[str, dict[str, Any]] = {}

    if args.config:
        sources = load_sources(args.config)
        for source in sources:
            stype = source.get("type", "")
            if stype == "typeshed":
                all_modules.update(process_typeshed_source(source))
            elif stype == "pyi_repo":
                all_modules.update(process_pyi_repo_source(source))
            elif stype == "pep561":
                all_modules.update(process_pep561_source(source))
            else:
                logger.warning("Unknown source type: %s", stype)

    elif args.typeshed:
        all_modules.update(process_typeshed_source({"path": str(args.typeshed)}))

    else:
        logger.error("Either --config or --typeshed is required")
        return 1

    if not all_modules:
        logger.error("No modules converted")
        return 1

    # Post-processing: MRO computation and inheritance flattening
    if not args.no_mro:
        from mro import flatten_inheritance

        logger.info("Computing MRO and flattening inheritance for %d modules...", len(all_modules))
        flatten_inheritance(all_modules)
        logger.info("MRO computation complete")

    manifest = generate_manifest(args.output, all_modules, args.base_url)
    stats = manifest["statistics"]
    logger.info(
        "Done: %d modules, %d functions, %d classes, %d constants, %d attributes",
        stats["total_modules"],
        stats["total_functions"],
        stats["total_classes"],
        stats["total_constants"],
        stats["total_attributes"],
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())
