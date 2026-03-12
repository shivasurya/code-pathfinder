"""C3 MRO computation and inheritance flattening for type registries.

Implements Python's C3 linearization algorithm to compute Method Resolution
Order (MRO) for classes, then flattens inherited methods/attributes into
each class so the Go runtime does zero MRO computation.

Also handles cross-package dependency resolution via METADATA.toml.
"""

from __future__ import annotations

import logging
from pathlib import Path
from typing import Any

logger = logging.getLogger(__name__)


# ── METADATA.toml Dependency Parsing ─────────────────────────────────────────


def parse_metadata_requires(metadata_path: Path) -> list[str]:
    """Parse 'requires' field from a typeshed METADATA.toml file.

    Returns importable package names extracted from dependency specifiers.
    E.g., ["types-requests", "urllib3>=2"] -> ["requests", "urllib3"]
    """
    if not metadata_path.exists():
        return []

    requires: list[str] = []
    for line in metadata_path.read_text(encoding="utf-8").splitlines():
        line = line.strip()
        if line.startswith("requires"):
            # Parse: requires = ["types-paramiko", "types-requests", "urllib3>=2"]
            _, _, value = line.partition("=")
            value = value.strip().strip("[]")
            for dep in value.split(","):
                dep = dep.strip().strip('"').strip("'")
                if not dep:
                    continue
                pkg = _normalize_dep_name(dep)
                if pkg:
                    requires.append(pkg)
            break
    return requires


def _normalize_dep_name(dep_specifier: str) -> str:
    """Normalize a dependency specifier to an importable package name.

    "types-requests" -> "requests"
    "urllib3>=2" -> "urllib3"
    "numpy>=1.20" -> "numpy"
    "cryptography>=37.0.0" -> "cryptography"
    """
    # Strip version specifiers
    for sep in (">=", "<=", "==", "!=", "~=", ">", "<"):
        dep_specifier = dep_specifier.split(sep)[0]
    name = dep_specifier.strip()

    # Strip "types-" prefix (typeshed convention)
    if name.startswith("types-"):
        name = name[len("types-"):]

    return name.lower().replace("-", "_")


# ── Dependency Graph & Topological Sort ──────────────────────────────────────


def build_dependency_graph(
    stubs_path: Path,
    package_names: list[str],
) -> dict[str, list[str]]:
    """Build dependency graph from METADATA.toml files.

    Args:
        stubs_path: Path to typeshed stubs/ directory.
        package_names: List of known package directory names.

    Returns:
        Dict mapping package_name -> [dependency_names].
    """
    # Build reverse map: importable_name -> directory_name
    dir_to_importable: dict[str, str] = {}
    for name in package_names:
        pkg_dir = stubs_path / name
        if pkg_dir.is_dir():
            # Check inner dir for importable name
            for child in pkg_dir.iterdir():
                if child.is_dir() and (child / "__init__.pyi").exists():
                    dir_to_importable[name.lower().replace("-", "_")] = name
                    dir_to_importable[child.name] = name
                    break
            else:
                dir_to_importable[name.lower().replace("-", "_")] = name

    graph: dict[str, list[str]] = {}
    for name in package_names:
        metadata = stubs_path / name / "METADATA.toml"
        deps = parse_metadata_requires(metadata)
        # Filter to only known packages
        resolved_deps = []
        for dep in deps:
            if dep in dir_to_importable:
                resolved_deps.append(dir_to_importable[dep])
            elif dep.replace("_", "-") in dir_to_importable:
                resolved_deps.append(dir_to_importable[dep.replace("_", "-")])
        graph[name] = resolved_deps

    return graph


def topological_sort(graph: dict[str, list[str]]) -> list[str]:
    """Topological sort of dependency graph.

    Returns packages in load order (dependencies first).
    Handles cycles by breaking them and logging a warning.
    """
    visited: set[str] = set()
    in_stack: set[str] = set()
    result: list[str] = []

    def visit(node: str) -> None:
        if node in visited:
            return
        if node in in_stack:
            logger.warning("Circular dependency detected involving: %s", node)
            return
        in_stack.add(node)
        for dep in graph.get(node, []):
            visit(dep)
        in_stack.discard(node)
        visited.add(node)
        result.append(node)

    for node in sorted(graph.keys()):
        visit(node)

    return result


# ── C3 Linearization ────────────────────────────────────────────────────────


def c3_linearize(
    class_fqn: str,
    class_registry: dict[str, dict[str, Any]],
) -> list[str]:
    """Compute MRO using C3 linearization.

    Args:
        class_fqn: Fully qualified class name (e.g., "requests.Session").
        class_registry: Flat map of class_fqn -> class_data with "bases" field.

    Returns:
        Ordered list of ancestor FQNs (including the class itself).
    """
    if class_fqn not in class_registry:
        return [class_fqn]

    cls_data = class_registry[class_fqn]
    bases = cls_data.get("bases", [])

    if not bases:
        return [class_fqn, "builtins.object"]

    # Compute MRO for each base
    base_mros: list[list[str]] = []
    for base in bases:
        base_mro = c3_linearize(base, class_registry)
        base_mros.append(base_mro)

    # Add the list of direct bases
    base_mros.append(list(bases))

    # C3 merge
    merged = _c3_merge(base_mros)
    if merged is None:
        logger.warning(
            "Inconsistent MRO for %s, falling back to left-to-right", class_fqn
        )
        merged = _fallback_mro(class_fqn, class_registry)
    else:
        merged = [class_fqn] + merged

    # Ensure builtins.object is at the end
    if "builtins.object" in merged:
        merged.remove("builtins.object")
    merged.append("builtins.object")

    return merged


def _c3_merge(sequences: list[list[str]]) -> list[str] | None:
    """C3 merge algorithm.

    Returns merged linearization, or None if inconsistent.
    """
    result: list[str] = []
    seqs = [list(s) for s in sequences if s]  # deep copy, filter empty

    max_iterations = 1000  # safety bound
    iterations = 0

    while seqs:
        iterations += 1
        if iterations > max_iterations:
            return None

        # Find a head that doesn't appear in the tail of any other sequence
        candidate = None
        for seq in seqs:
            head = seq[0]
            # Check if head is NOT in any tail
            in_tail = False
            for other_seq in seqs:
                if head in other_seq[1:]:
                    in_tail = True
                    break
            if not in_tail:
                candidate = head
                break

        if candidate is None:
            # No valid candidate — inconsistent hierarchy
            return None

        result.append(candidate)

        # Remove candidate from all sequences
        new_seqs: list[list[str]] = []
        for seq in seqs:
            filtered = [x for x in seq if x != candidate]
            if filtered:
                new_seqs.append(filtered)
        seqs = new_seqs

    return result


def _fallback_mro(
    class_fqn: str,
    class_registry: dict[str, dict[str, Any]],
) -> list[str]:
    """Simple left-to-right DFS fallback for inconsistent MROs."""
    visited: set[str] = set()
    result: list[str] = [class_fqn]
    visited.add(class_fqn)

    stack = list(class_registry.get(class_fqn, {}).get("bases", []))
    while stack:
        current = stack.pop(0)
        if current in visited or current == "builtins.object":
            continue
        visited.add(current)
        result.append(current)
        cls_data = class_registry.get(current, {})
        stack.extend(cls_data.get("bases", []))

    return result


# ── Class Registry Builder ──────────────────────────────────────────────────


def build_class_registry(
    modules: dict[str, dict[str, Any]],
) -> dict[str, dict[str, Any]]:
    """Build a flat class registry from all module data.

    Maps fully qualified class name -> class data.
    E.g., "requests.Response" -> {"type": "class", "methods": {...}, ...}
    """
    registry: dict[str, dict[str, Any]] = {}

    for mod_name, mod_data in modules.items():
        for cls_name, cls_data in mod_data.get("classes", {}).items():
            fqn = f"{mod_name}.{cls_name}"
            registry[fqn] = cls_data

    return registry


# ── Inheritance Flattening ──────────────────────────────────────────────────


def flatten_inheritance(
    modules: dict[str, dict[str, Any]],
    class_registry: dict[str, dict[str, Any]] | None = None,
) -> dict[str, dict[str, Any]]:
    """Compute MRO and flatten inherited members into each class.

    For each class:
    1. Compute C3 MRO
    2. Walk ancestors in reverse (most base first)
    3. Copy methods/attributes not overridden by the class
    4. Mark with inherited_from field

    Args:
        modules: Module data dict (modified in place).
        class_registry: Optional pre-built class registry.

    Returns:
        The modified modules dict.
    """
    if class_registry is None:
        class_registry = build_class_registry(modules)

    for mod_name, mod_data in modules.items():
        for cls_name, cls_data in mod_data.get("classes", {}).items():
            fqn = f"{mod_name}.{cls_name}"
            mro = c3_linearize(fqn, class_registry)

            cls_data["mro"] = mro

            inherited_methods: dict[str, Any] = {}
            inherited_attributes: dict[str, Any] = {}

            # Walk MRO in reverse (most base first), skip self and builtins.object
            ancestors = [a for a in reversed(mro) if a != fqn and a != "builtins.object"]

            for ancestor_fqn in ancestors:
                ancestor = class_registry.get(ancestor_fqn)
                if ancestor is None:
                    continue

                # Copy methods not overridden
                for method_name, method_data in ancestor.get("methods", {}).items():
                    if method_name not in cls_data.get("methods", {}):
                        entry = dict(method_data)
                        entry["inherited_from"] = ancestor_fqn
                        inherited_methods[method_name] = entry

                # Copy attributes not overridden
                for attr_name, attr_data in ancestor.get("attributes", {}).items():
                    if attr_name not in cls_data.get("attributes", {}):
                        entry = dict(attr_data)
                        entry["inherited_from"] = ancestor_fqn
                        inherited_attributes[attr_name] = entry

            if inherited_methods:
                cls_data["inherited_methods"] = inherited_methods
            if inherited_attributes:
                cls_data["inherited_attributes"] = inherited_attributes

    return modules
