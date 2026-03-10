#!/usr/bin/env python3
"""
Generic Python stdlib registry generator.

This script introspects Python standard library modules and generates
JSON registries containing functions, classes, constants, and attributes
with type information for static analysis.

Minimum Python version: 3.9

Usage:
    # Generate ALL stdlib modules for Python 3.14
    python3.14 generate_stdlib_registry.py --all --output-dir ./registries/python3.14/stdlib/v1/

    # Generate specific modules
    python generate_stdlib_registry.py --modules os,sys,pathlib --output-dir ./output/

    # Generate from config
    python generate_stdlib_registry.py --config high-impact.yaml --output-dir ./output/
"""

import argparse
import ast
import hashlib
import importlib
import inspect
import json
import os
import pkgutil
import re
import sys
import sysconfig
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Dict, List, Optional


def get_type_name(annotation) -> str:
    """Convert type annotation to fully qualified name."""
    if annotation is None:
        return "unknown"

    # Handle string annotations
    if isinstance(annotation, str):
        return annotation

    # Handle typing module types
    if hasattr(annotation, "__module__") and hasattr(annotation, "__name__"):
        module = annotation.__module__
        name = annotation.__name__

        # Normalize builtins
        if module == "builtins":
            return f"builtins.{name}"
        return f"{module}.{name}"

    # Handle generic types (List[str], Dict[str, int], etc.)
    if hasattr(annotation, "__origin__"):
        origin = annotation.__origin__
        origin_name = get_type_name(origin)

        # Get type arguments if available
        if hasattr(annotation, "__args__"):
            args = annotation.__args__
            if args:
                arg_names = [get_type_name(arg) for arg in args]
                return f"{origin_name}[{', '.join(arg_names)}]"

        return origin_name

    # Fallback to string representation
    return str(annotation)


def clean_docstring(docstring: str) -> str:
    """Clean and normalize docstring."""
    if not docstring:
        return ""

    # Remove leading/trailing whitespace
    lines = docstring.strip().split("\n")

    # Remove common indentation
    min_indent = float("inf")
    for line in lines[1:]:  # Skip first line
        stripped = line.lstrip()
        if stripped:
            indent = len(line) - len(stripped)
            min_indent = min(min_indent, indent)

    if min_indent != float("inf"):
        lines = [lines[0]] + [line[min_indent:] if len(line) > min_indent else line
                              for line in lines[1:]]

    # Join and truncate if too long
    cleaned = "\n".join(lines).strip()
    if len(cleaned) > 500:
        cleaned = cleaned[:497] + "..."

    return cleaned


def infer_from_docstring(func) -> str:
    """Infer return type from docstring patterns."""
    if not func.__doc__:
        return "unknown"

    doc = func.__doc__.lower()

    # Common patterns in docstrings
    patterns = {
        "return": ["str", "int", "float", "bool", "list", "dict", "tuple", "set"],
    }

    for keyword, types in patterns.items():
        if keyword in doc:
            for type_name in types:
                if type_name in doc:
                    return f"builtins.{type_name}"

    return "unknown"


def find_typeshed_path() -> Optional[Path]:
    """Find the typeshed stdlib stubs directory.

    Searches in order:
    1. Bundled typeshed in mypy installation
    2. typeshed-client package
    3. System typeshed (e.g., /usr/lib/python3/typeshed)
    4. Local typeshed checkout specified by TYPESHED_PATH env var
    """
    # Check env var first
    env_path = os.environ.get("TYPESHED_PATH")
    if env_path:
        p = Path(env_path) / "stdlib"
        if p.exists():
            return p

    # Check mypy's bundled typeshed
    try:
        import mypy.typeshed  # type: ignore
        p = Path(mypy.typeshed.__path__[0]) / "stdlib"
        if p.exists():
            return p
    except (ImportError, AttributeError):
        pass

    # Check common system locations
    for candidate in [
        Path("/usr/lib/python3/typeshed/stdlib"),
        Path("/usr/share/typeshed/stdlib"),
        Path(sys.prefix) / "lib" / "typeshed" / "stdlib",
    ]:
        if candidate.exists():
            return candidate

    return None


_stub_parse_stack: set = set()  # Recursion guard for cross-module stub resolution

def parse_typeshed_stub(module_name: str, typeshed_path: Optional[Path]) -> Dict[str, Dict[str, str]]:
    """Parse a typeshed .pyi stub file to extract return types and class bases.

    Returns a dict like:
    {
        "functions": {"func_name": "return.Type", ...},
        "classes": {
            "ClassName": {
                "__bases__": ["BaseClass1", "BaseClass2"],
                "method_name": "return.Type",
                ...
            }
        }
    }
    """
    result: Dict[str, Any] = {"functions": {}, "classes": {}}

    if typeshed_path is None:
        return result

    # Recursion guard: prevent infinite loops in cross-module stub resolution
    if module_name in _stub_parse_stack:
        return result
    _stub_parse_stack.add(module_name)

    # Find the stub file
    # Module can be a package (os/__init__.pyi) or a simple module (json.pyi)
    parts = module_name.split(".")
    candidates = [
        typeshed_path / (module_name.replace(".", "/") + ".pyi"),
        typeshed_path / module_name.replace(".", "/") / "__init__.pyi",
    ]
    # For top-level like "sqlite3", also check VERSIONS file subdir
    if len(parts) == 1:
        candidates.append(typeshed_path / parts[0] / "__init__.pyi")
        candidates.append(typeshed_path / (parts[0] + ".pyi"))

    stub_path = None
    for c in candidates:
        if c.exists():
            stub_path = c
            break

    if stub_path is None:
        return result

    try:
        source = stub_path.read_text(encoding="utf-8")
        tree = ast.parse(source, filename=str(stub_path))
    except (SyntaxError, UnicodeDecodeError):
        return result

    def resolve_annotation(node) -> str:
        """Convert an AST annotation node to a string type name."""
        if node is None:
            return "unknown"
        if isinstance(node, ast.Constant):
            if isinstance(node.value, str):
                return node.value
            return str(node.value)
        if isinstance(node, ast.Name):
            return node.id
        if isinstance(node, ast.Attribute):
            parts = []
            current = node
            while isinstance(current, ast.Attribute):
                parts.append(current.attr)
                current = current.value
            if isinstance(current, ast.Name):
                parts.append(current.id)
            return ".".join(reversed(parts))
        if isinstance(node, ast.Subscript):
            # e.g., List[str], Optional[int]
            base = resolve_annotation(node.value)
            return base  # Strip generics for now — we want the base type
        if isinstance(node, ast.BinOp) and isinstance(node.op, ast.BitOr):
            # Union type (X | Y) — take first
            return resolve_annotation(node.left)
        return "unknown"

    def is_typevar(type_name: str) -> bool:
        """Check if a type name looks like a TypeVar (e.g., _CursorT, _T)."""
        return (type_name.startswith("_") and len(type_name) > 1
                and type_name[1:2].isupper() and "." not in type_name)

    def qualify_type(type_name: str) -> str:
        """Qualify a short type name with its module if known."""
        builtin_types = {"str", "int", "float", "bool", "list", "dict", "tuple", "set",
                         "bytes", "bytearray", "None", "type", "object", "complex",
                         "frozenset", "memoryview", "range", "slice", "property"}
        if type_name in builtin_types:
            if type_name == "None":
                return "builtins.NoneType"
            return f"builtins.{type_name}"
        # Normalize internal module prefixes (e.g., _socket.socket -> socket.socket)
        if "." in type_name:
            parts = type_name.split(".", 1)
            if parts[0].startswith("_") and not parts[0].startswith("__"):
                type_name = parts[0].lstrip("_") + "." + parts[1]
        # If it looks like a simple class name within same module, qualify it
        if "." not in type_name and type_name[0:1].isupper():
            return f"{module_name}.{type_name}"
        return type_name

    # First pass: collect import aliases (e.g., from _hashlib import openssl_md5 as md5)
    # Maps alias_name -> (source_module, original_name)
    import_aliases: Dict[str, tuple] = {}
    for node in ast.iter_child_nodes(tree):
        if isinstance(node, ast.ImportFrom) and node.module:
            src_module = node.module
            for alias in node.names:
                if alias.asname:
                    import_aliases[alias.asname] = (src_module, alias.name)

    def process_node(node):
        """Process a single AST node (function, class, or if-block)."""
        if isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef)):
            ret = resolve_annotation(node.returns)
            if ret != "unknown" and not is_typevar(ret):
                # Keep first definition (don't overwrite overloads)
                if node.name not in result["functions"]:
                    result["functions"][node.name] = qualify_type(ret)

        elif isinstance(node, ast.ClassDef):
            class_info: Dict[str, str] = {}

            # Extract bases
            bases = []
            for base in node.bases:
                base_name = resolve_annotation(base)
                if base_name != "unknown":
                    bases.append(qualify_type(base_name))
            if bases:
                class_info["__bases__"] = bases  # type: ignore

            # Extract method return types (keep first non-TypeVar for overloads)
            for item in ast.iter_child_nodes(node):
                if isinstance(item, (ast.FunctionDef, ast.AsyncFunctionDef)):
                    ret = resolve_annotation(item.returns)
                    if item.name not in class_info and ret != "unknown" and not is_typevar(ret):
                        class_info[item.name] = qualify_type(ret)
                # Recurse into if-blocks inside classes too
                elif isinstance(item, ast.If):
                    for sub in item.body + item.orelse:
                        if isinstance(sub, (ast.FunctionDef, ast.AsyncFunctionDef)):
                            ret = resolve_annotation(sub.returns)
                            if sub.name not in class_info and ret != "unknown" and not is_typevar(ret):
                                class_info[sub.name] = qualify_type(ret)

            result["classes"][node.name] = class_info

        elif isinstance(node, ast.If):
            # Recurse into if/else blocks (handles `if sys.version_info >= ...`)
            for sub in node.body + node.orelse:
                process_node(sub)

    # Process top-level definitions
    for node in ast.iter_child_nodes(tree):
        process_node(node)

    # Second pass: resolve import aliases by parsing the source module's stub
    # e.g., "from _hashlib import openssl_md5 as md5" → parse _hashlib.pyi → openssl_md5() -> HASH
    for alias_name, (src_module, original_name) in import_aliases.items():
        if alias_name not in result["functions"]:
            # Parse the source module's stub to find the original function's return type
            src_stub = parse_typeshed_stub(src_module, typeshed_path)
            if original_name in src_stub.get("functions", {}):
                ret_type = src_stub["functions"][original_name]
                # Normalize the return type module prefix
                if "." in ret_type:
                    parts = ret_type.split(".", 1)
                    if parts[0].startswith("_") and not parts[0].startswith("__"):
                        ret_type = parts[0].lstrip("_") + "." + parts[1]
                result["functions"][alias_name] = ret_type

    _stub_parse_stack.discard(module_name)
    return result


def introspect_function(func) -> Dict[str, Any]:
    """Extract function signature and return type."""
    result = {
        "return_type": "unknown",
        "confidence": 0.5,
        "params": [],
        "source": "heuristic"
    }

    # Try to get signature (works for most functions)
    try:
        sig = inspect.signature(func)

        # Extract parameters
        for param_name, param in sig.parameters.items():
            param_info = {
                "name": param_name,
                "type": "unknown",
                "required": param.default == inspect.Parameter.empty
            }

            # Type annotation
            if param.annotation != inspect.Parameter.empty:
                param_info["type"] = get_type_name(param.annotation)

            result["params"].append(param_info)

        # Extract return type annotation
        if sig.return_annotation != inspect.Signature.empty:
            result["return_type"] = get_type_name(sig.return_annotation)
            result["confidence"] = 1.0
            result["source"] = "annotation"

    except (ValueError, TypeError):
        # Builtins often don't have inspectable signatures
        # Use docstring or manual mappings
        result["return_type"] = infer_from_docstring(func)
        result["confidence"] = 0.7
        result["source"] = "docstring"

    # Add docstring
    if func.__doc__:
        result["docstring"] = clean_docstring(func.__doc__)

    return result


def introspect_class(cls, module_name: str = "") -> Dict[str, Any]:
    """Extract class methods, bases, MRO, and their signatures."""
    result = {
        "type": "class",
        "methods": {},
        "bases": [],
        "mro": []
    }

    # Extract base classes (fully qualified names)
    try:
        for base in cls.__bases__:
            if base is object:
                result["bases"].append("object")
            else:
                base_module = getattr(base, "__module__", "")
                base_name = getattr(base, "__qualname__", getattr(base, "__name__", str(base)))
                # Normalize internal modules (e.g., _io -> io)
                if base_module.startswith("_") and not base_module.startswith("__"):
                    base_module = base_module.lstrip("_")
                if base_module and base_module != "builtins":
                    result["bases"].append(f"{base_module}.{base_name}")
                else:
                    result["bases"].append(base_name)
    except Exception:
        pass

    # Extract MRO (method resolution order)
    try:
        for mro_cls in cls.__mro__:
            mro_module = getattr(mro_cls, "__module__", "")
            mro_name = getattr(mro_cls, "__qualname__", getattr(mro_cls, "__name__", str(mro_cls)))
            # Normalize internal modules
            if mro_module.startswith("_") and not mro_module.startswith("__"):
                mro_module = mro_module.lstrip("_")
            if mro_module and mro_module != "builtins":
                result["mro"].append(f"{mro_module}.{mro_name}")
            elif mro_cls is object:
                result["mro"].append("object")
            else:
                result["mro"].append(mro_name)
    except Exception:
        pass

    # Add docstring
    if cls.__doc__:
        result["docstring"] = clean_docstring(cls.__doc__)

    # Extract methods (including builtin methods that may not be functions)
    for name in dir(cls):
        # Skip private methods except important ones
        if name.startswith("_") and name not in ["__init__", "__call__", "__enter__", "__exit__", "__iter__", "__next__"]:
            continue

        try:
            attr = getattr(cls, name)
            if inspect.ismethod(attr) or inspect.isfunction(attr) or inspect.ismethoddescriptor(attr) or inspect.isbuiltin(attr):
                result["methods"][name] = introspect_function(attr)
        except Exception:
            # Skip problematic methods
            pass

    return result


def introspect_constant(obj: Any) -> Dict[str, Any]:
    """Extract constant type and value."""
    obj_type = type(obj)
    type_name = f"builtins.{obj_type.__name__}"

    # Get string representation
    value_repr = repr(obj)
    if len(value_repr) > 100:
        value_repr = value_repr[:97] + "..."

    return {
        "type": type_name,
        "value": value_repr,
        "confidence": 1.0
    }


def introspect_attribute(obj: Any) -> Dict[str, Any]:
    """Extract module-level attribute (e.g., os.environ, sys.modules)."""
    obj_type = type(obj)

    # Get fully qualified type name
    if hasattr(obj_type, "__module__") and hasattr(obj_type, "__name__"):
        type_fqn = f"{obj_type.__module__}.{obj_type.__name__}"
    else:
        type_fqn = str(obj_type)

    # Detect behavior patterns
    behaves_like = None
    confidence = 0.7

    if hasattr(obj, "keys") and hasattr(obj, "values") and hasattr(obj, "__getitem__"):
        behaves_like = "builtins.dict"
        confidence = 0.9
    elif hasattr(obj, "__iter__") and not isinstance(obj, str):
        if hasattr(obj, "append"):
            behaves_like = "builtins.list"
            confidence = 0.9

    result = {
        "type": type_fqn,
        "confidence": confidence
    }

    if behaves_like:
        result["behaves_like"] = behaves_like

    # Add docstring if available
    if hasattr(obj, "__doc__") and obj.__doc__:
        result["docstring"] = clean_docstring(obj.__doc__)

    return result


def apply_typeshed_overlay(result: Dict[str, Any], module_name: str, typeshed_path: Optional[Path], verbose: bool = False) -> None:
    """Overlay typeshed stub data onto introspected module data.

    Patches in return types for functions/methods where runtime introspection
    returned "unknown" (common for C builtins like sqlite3, hashlib).
    Also patches in class bases from stubs when runtime bases use internal names.
    """
    stub_data = parse_typeshed_stub(module_name, typeshed_path)
    if not stub_data["functions"] and not stub_data["classes"]:
        return

    patched = 0

    # Patch function return types
    for func_name, return_type in stub_data["functions"].items():
        if func_name in result["functions"]:
            func = result["functions"][func_name]
            if func.get("return_type") == "unknown" or func.get("source") in ("heuristic", "docstring"):
                func["return_type"] = return_type
                func["confidence"] = 0.95
                func["source"] = "typeshed"
                patched += 1

    # Patch class method return types and bases
    for class_name, class_info in stub_data["classes"].items():
        if class_name not in result["classes"]:
            continue

        cls_result = result["classes"][class_name]

        # Patch bases from typeshed if they provide better qualified names
        stub_bases = class_info.get("__bases__")
        if stub_bases:
            # Prefer typeshed bases — they have proper public names (e.g., io.TextIOBase vs io._TextIOBase)
            cls_result["bases"] = list(stub_bases)

        # Patch method return types
        for method_name, return_type in class_info.items():
            if method_name.startswith("__") and method_name != "__init__":
                continue
            if method_name in cls_result.get("methods", {}):
                method = cls_result["methods"][method_name]
                if method.get("return_type") == "unknown" or method.get("source") in ("heuristic", "docstring"):
                    method["return_type"] = return_type
                    method["confidence"] = 0.95
                    method["source"] = "typeshed"
                    patched += 1

    if verbose and patched > 0:
        print(f"  [typeshed] Patched {patched} return types for {module_name}")


def introspect_module(module_name: str, typeshed_path: Optional[Path] = None, verbose: bool = False) -> Optional[Dict[str, Any]]:
    """Generic introspection for ANY stdlib module.

    Combines runtime introspection with typeshed stub data for complete
    type information including return types for C builtins and class MRO.
    """
    try:
        module = importlib.import_module(module_name)
    except Exception as e:
        print(f"Error: Failed to import {module_name}: {e}", file=sys.stderr)
        return None

    result = {
        "module": module_name,
        "python_version": f"{sys.version_info.major}.{sys.version_info.minor}.{sys.version_info.micro}",
        "generated_at": datetime.now(timezone.utc).isoformat().replace("+00:00", "Z"),
        "functions": {},
        "classes": {},
        "constants": {},
        "attributes": {}
    }

    # Introspect all public members
    for name in dir(module):
        # Skip private members
        if name.startswith("_"):
            continue

        try:
            obj = getattr(module, name)

            # Skip imported modules
            if inspect.ismodule(obj):
                continue

            # Categorize and introspect
            if inspect.isfunction(obj) or inspect.isbuiltin(obj):
                result["functions"][name] = introspect_function(obj)
            elif inspect.isclass(obj):
                result["classes"][name] = introspect_class(obj, module_name)
            elif isinstance(obj, (str, int, float, bool, tuple)):
                result["constants"][name] = introspect_constant(obj)
            else:
                # Module-level attributes (os.environ, sys.modules, etc.)
                result["attributes"][name] = introspect_attribute(obj)

        except Exception as e:
            # Log but don't fail on individual members
            print(f"Warning: Failed to introspect {module_name}.{name}: {e}", file=sys.stderr)

    # Apply typeshed overlay for C builtins missing return types
    if typeshed_path:
        apply_typeshed_overlay(result, module_name, typeshed_path, verbose)

    return result


def generate_manifest(output_dir: Path, python_version: tuple, modules: List[str]) -> Dict[str, Any]:
    """Generate manifest.json with registry metadata."""
    base_url = f"https://assets.codepathfinder.dev/registries/python{python_version[0]}.{python_version[1]}/stdlib/v1"

    manifest = {
        "schema_version": "1.0.0",
        "registry_version": "v1",
        "python_version": {
            "major": python_version[0],
            "minor": python_version[1],
            "patch": python_version[2],
            "full": f"{python_version[0]}.{python_version[1]}.{python_version[2]}"
        },
        "generated_at": datetime.now(timezone.utc).isoformat().replace("+00:00", "Z"),
        "generator_version": "1.0.0",
        "base_url": base_url,
        "modules": []
    }

    # Add module entries
    for module_name in sorted(modules):
        file_name = f"{module_name}_stdlib.json"
        file_path = output_dir / file_name

        if not file_path.exists():
            continue

        # Calculate checksum and size
        with open(file_path, "rb") as f:
            content = f.read()
            checksum = hashlib.sha256(content).hexdigest()
            size_bytes = len(content)

        manifest["modules"].append({
            "name": module_name,
            "file": file_name,
            "url": f"{base_url}/{file_name}",
            "size_bytes": size_bytes,
            "checksum": f"sha256:{checksum}"
        })

    # Calculate statistics
    total_funcs = total_classes = total_constants = total_attrs = 0
    for module_info in manifest["modules"]:
        module_path = output_dir / module_info["file"]
        with open(module_path) as f:
            data = json.load(f)
            total_funcs += len(data.get("functions", {}))
            total_classes += len(data.get("classes", {}))
            total_constants += len(data.get("constants", {}))
            total_attrs += len(data.get("attributes", {}))

    manifest["statistics"] = {
        "total_modules": len(manifest["modules"]),
        "total_functions": total_funcs,
        "total_classes": total_classes,
        "total_constants": total_constants,
        "total_attributes": total_attrs
    }

    return manifest


def get_all_stdlib_modules() -> List[str]:
    """Get list of all public stdlib modules.

    For Python 3.10+, uses sys.stdlib_module_names (PEP 585).
    For Python 3.9, uses pkgutil and sysconfig to discover modules from:
    - Built-in modules (C-level)
    - Standard library path (pure Python modules)
    - lib-dynload / DLLs (compiled extension modules)
    - Platform-specific stdlib path
    """
    # Python 3.10+ has sys.stdlib_module_names
    if hasattr(sys, "stdlib_module_names"):
        # Filter out private modules
        return sorted([m for m in sys.stdlib_module_names if not m.startswith("_")])

    # Fallback for Python 3.9: Use pkgutil to discover stdlib modules
    stdlib_modules = set()

    # Add built-in modules
    stdlib_modules.update(sys.builtin_module_names)

    # Get stdlib path
    stdlib_path = sysconfig.get_path('stdlib')
    if stdlib_path:
        # Discover modules in stdlib path
        try:
            for importer, modname, ispkg in pkgutil.iter_modules([stdlib_path]):
                stdlib_modules.add(modname)
        except Exception as e:
            print(f"Warning: Error discovering modules from {stdlib_path}: {e}", file=sys.stderr)

        # Also check lib-dynload subdirectory for compiled extension modules (Unix/Linux)
        # and DLLs directory (Windows)
        for dynload_dir_name in ['lib-dynload', 'DLLs']:
            dynload_path = os.path.join(stdlib_path, dynload_dir_name)
            if os.path.exists(dynload_path):
                try:
                    for importer, modname, ispkg in pkgutil.iter_modules([dynload_path]):
                        stdlib_modules.add(modname)
                except Exception as e:
                    print(f"Warning: Error discovering modules from {dynload_path}: {e}", file=sys.stderr)

    # Get platstdlib path (platform-specific stdlib modules)
    platstdlib_path = sysconfig.get_path('platstdlib')
    if platstdlib_path and platstdlib_path != stdlib_path:
        try:
            for importer, modname, ispkg in pkgutil.iter_modules([platstdlib_path]):
                stdlib_modules.add(modname)
        except Exception as e:
            print(f"Warning: Error discovering modules from {platstdlib_path}: {e}", file=sys.stderr)

    # Filter out private modules and return sorted list
    return sorted([m for m in stdlib_modules if not m.startswith("_")])


def main():
    """Main entry point."""
    parser = argparse.ArgumentParser(
        description="Generate Python stdlib registry for static analysis",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog=__doc__
    )

    parser.add_argument(
        "--all",
        action="store_true",
        help="Generate registries for ALL stdlib modules"
    )

    parser.add_argument(
        "--modules",
        type=str,
        help="Comma-separated list of modules to generate (e.g., os,sys,pathlib)"
    )

    parser.add_argument(
        "--output-dir",
        type=str,
        required=True,
        help="Output directory for generated registries"
    )

    parser.add_argument(
        "--verbose",
        action="store_true",
        help="Enable verbose logging"
    )

    parser.add_argument(
        "--typeshed-path",
        type=str,
        help="Path to typeshed stdlib stubs directory (auto-detected if not specified)"
    )

    args = parser.parse_args()

    # Check Python version
    if sys.version_info < (3, 9):
        print(f"Error: Python 3.9 or higher is required. Current version: {sys.version_info.major}.{sys.version_info.minor}.{sys.version_info.micro}", file=sys.stderr)
        return 1

    # Find typeshed path
    typeshed_path = None
    if args.typeshed_path:
        typeshed_path = Path(args.typeshed_path)
        # Normalize: if user passed the typeshed root (containing stdlib/), append stdlib
        if (typeshed_path / "stdlib").exists():
            typeshed_path = typeshed_path / "stdlib"
        if not typeshed_path.exists():
            print(f"Warning: Specified typeshed path does not exist: {typeshed_path}", file=sys.stderr)
            typeshed_path = None
    else:
        typeshed_path = find_typeshed_path()

    if typeshed_path:
        print(f"Using typeshed stubs from: {typeshed_path}")
    else:
        print("Warning: No typeshed stubs found. C builtin return types may be incomplete.", file=sys.stderr)
        print("  Install mypy (pip install mypy) or set TYPESHED_PATH environment variable.", file=sys.stderr)

    # Determine which modules to generate
    if args.all:
        modules = get_all_stdlib_modules()
        detection_method = "sys.stdlib_module_names" if hasattr(sys, "stdlib_module_names") else "pkgutil fallback"
        if args.verbose:
            print(f"Using {detection_method} to discover stdlib modules")
        print(f"Generating registries for ALL {len(modules)} stdlib modules...")
    elif args.modules:
        modules = [m.strip() for m in args.modules.split(",")]
        print(f"Generating registries for {len(modules)} modules: {', '.join(modules)}")
    else:
        parser.error("Must specify either --all or --modules")

    # Create output directory
    output_dir = Path(args.output_dir)
    output_dir.mkdir(parents=True, exist_ok=True)

    # Generate registries
    successful = []
    failed = []

    for i, module_name in enumerate(modules, 1):
        if args.verbose:
            print(f"[{i}/{len(modules)}] Processing {module_name}...", flush=True)

        # Introspect module with typeshed overlay
        data = introspect_module(module_name, typeshed_path, args.verbose)

        if data is None:
            failed.append(module_name)
            continue

        # Write to file
        output_file = output_dir / f"{module_name}_stdlib.json"
        with open(output_file, "w") as f:
            json.dump(data, f, indent=2)

        successful.append(module_name)

        if args.verbose:
            print(f"  ✓ Functions: {len(data['functions'])}, "
                  f"Classes: {len(data['classes'])}, "
                  f"Constants: {len(data['constants'])}, "
                  f"Attributes: {len(data['attributes'])}")

    # Generate manifest
    print(f"\nGenerating manifest.json...")
    manifest = generate_manifest(output_dir, sys.version_info[:3], successful)

    manifest_file = output_dir / "manifest.json"
    with open(manifest_file, "w") as f:
        json.dump(manifest, f, indent=2)

    # Print summary
    print(f"\n{'='*60}")
    print(f"Registry Generation Complete")
    print(f"{'='*60}")
    print(f"Python version: {manifest['python_version']['full']}")
    print(f"Output directory: {output_dir}")
    print(f"Successful: {len(successful)} modules")
    print(f"Failed: {len(failed)} modules")
    print(f"\nStatistics:")
    print(f"  Total modules: {manifest['statistics']['total_modules']}")
    print(f"  Total functions: {manifest['statistics']['total_functions']}")
    print(f"  Total classes: {manifest['statistics']['total_classes']}")
    print(f"  Total constants: {manifest['statistics']['total_constants']}")
    print(f"  Total attributes: {manifest['statistics']['total_attributes']}")

    if failed:
        print(f"\nFailed modules ({len(failed)}):")
        for module_name in failed:
            print(f"  - {module_name}")

    print(f"{'='*60}\n")

    # Platform-specific modules that are expected to fail on certain platforms
    WINDOWS_ONLY_MODULES = {'msvcrt', 'nt', 'winreg', 'winsound', '_winapi', 'msilib'}
    UNIX_SPECIFIC_MODULES = {'nis', 'grp', 'pwd', 'resource', 'syslog', 'termios', 'tty'}
    MACOS_SPECIFIC_MODULES = {'_scproxy'}
    # Modules that may not be available in all builds or require display server
    OPTIONAL_MODULES = {'readline', 'dbm', 'gdbm', 'ossaudiodev', 'spwd'}
    GUI_MODULES = {'tkinter', 'turtle', 'turtledemo', 'idlelib'}
    DEPRECATED_MODULES = {'lib2to3'}
    PLATFORM_SPECIFIC_MODULES = WINDOWS_ONLY_MODULES | UNIX_SPECIFIC_MODULES | MACOS_SPECIFIC_MODULES | OPTIONAL_MODULES | GUI_MODULES | DEPRECATED_MODULES

    # Check if any non-platform-specific modules failed
    unexpected_failures = [m for m in failed if m not in PLATFORM_SPECIFIC_MODULES]

    if unexpected_failures:
        print(f"ERROR: Unexpected module failures: {', '.join(unexpected_failures)}")
        return 1

    return 0


if __name__ == "__main__":
    sys.exit(main())
