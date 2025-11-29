#!/usr/bin/env python3
"""
Generic Python stdlib registry generator.

This script introspects Python standard library modules and generates
JSON registries containing functions, classes, constants, and attributes
with type information for static analysis.

Usage:
    # Generate ALL stdlib modules for Python 3.14
    python3.14 generate_stdlib_registry.py --all --output-dir ./registries/python3.14/stdlib/v1/

    # Generate specific modules
    python generate_stdlib_registry.py --modules os,sys,pathlib --output-dir ./output/

    # Generate from config
    python generate_stdlib_registry.py --config high-impact.yaml --output-dir ./output/
"""

import argparse
import hashlib
import importlib
import inspect
import json
import os
import sys
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


def introspect_class(cls) -> Dict[str, Any]:
    """Extract class methods and their signatures."""
    result = {
        "type": "class",
        "methods": {}
    }

    # Add docstring
    if cls.__doc__:
        result["docstring"] = clean_docstring(cls.__doc__)

    # Extract methods
    for name in dir(cls):
        # Skip private methods except important ones
        if name.startswith("_") and name not in ["__init__", "__call__", "__enter__", "__exit__", "__iter__", "__next__"]:
            continue

        try:
            attr = getattr(cls, name)
            if inspect.ismethod(attr) or inspect.isfunction(attr):
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


def introspect_module(module_name: str) -> Optional[Dict[str, Any]]:
    """Generic introspection for ANY stdlib module."""
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
                result["classes"][name] = introspect_class(obj)
            elif isinstance(obj, (str, int, float, bool, tuple)):
                result["constants"][name] = introspect_constant(obj)
            else:
                # Module-level attributes (os.environ, sys.modules, etc.)
                result["attributes"][name] = introspect_attribute(obj)

        except Exception as e:
            # Log but don't fail on individual members
            print(f"Warning: Failed to introspect {module_name}.{name}: {e}", file=sys.stderr)

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
    """Get list of all public stdlib modules."""
    if not hasattr(sys, "stdlib_module_names"):
        raise RuntimeError("sys.stdlib_module_names not available (Python 3.10+ required)")

    # Filter out private modules
    return sorted([m for m in sys.stdlib_module_names if not m.startswith("_")])


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

    args = parser.parse_args()

    # Determine which modules to generate
    if args.all:
        modules = get_all_stdlib_modules()
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

        # Introspect module
        data = introspect_module(module_name)

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
    UNIX_SPECIFIC_MODULES = {'nis'}  # Network Information Service (may not be available on all systems)
    PLATFORM_SPECIFIC_MODULES = WINDOWS_ONLY_MODULES | UNIX_SPECIFIC_MODULES

    # Check if any non-platform-specific modules failed
    unexpected_failures = [m for m in failed if m not in PLATFORM_SPECIFIC_MODULES]

    if unexpected_failures:
        print(f"ERROR: Unexpected module failures: {', '.join(unexpected_failures)}")
        return 1

    return 0


if __name__ == "__main__":
    sys.exit(main())
