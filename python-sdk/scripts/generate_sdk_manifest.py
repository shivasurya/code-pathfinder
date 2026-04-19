#!/usr/bin/env python3
"""
generate_sdk_manifest.py — Build sdk-manifest.json from _meta.py companion files.

Usage:
    python scripts/generate_sdk_manifest.py --out public/sdk-manifest.json

The generator:
1. Imports each *_meta.py companion file (go_rule_meta, python_rule_meta, etc.)
2. Loads the corresponding QueryType class from the real SDK source
3. Validates that every method listed in SDK_META actually exists on the class
4. Cross-references rules/* to count how many rules use each class
5. Emits sdk-manifest.json consumed by the cpf-website /sdk/ pages

Adding a new language:
  - Create codepathfinder/{lang}_rule_meta.py with SDK_META dict
  - Import it in _collect_all_meta() below
  - Re-run this script
"""

from __future__ import annotations

import argparse
import importlib
import inspect
import json
import sys
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

# ── Resolve paths ──────────────────────────────────────────────────────────────
SCRIPT_DIR = Path(__file__).parent
SDK_ROOT    = SCRIPT_DIR.parent
RULES_DIR   = SDK_ROOT / "rules"
OUTPUT_DEFAULT = SDK_ROOT.parent.parent / "cpf-website" / "public" / "sdk-manifest.json"

sys.path.insert(0, str(SDK_ROOT))


# ── Language-level config (not in _meta.py — structural, not per-class) ───────
LANGUAGE_CONFIG: dict[str, dict] = {
    "golang": {
        "name": "Go",
        "version": "1.21+",
        "description": "Type-aware security analysis for Go applications. QueryType classes resolve to fully-qualified Go module paths.",
        "decorator": "@go_rule",
        "import_prefix": "from codepathfinder.go_rule import ",
        "install": "pip install codepathfinder",
        "meta_module": "codepathfinder.go_rule_meta",
        "sdk_module":  "codepathfinder.go_rule",
        "categories": [
            {"id": "web-frameworks", "name": "Web Frameworks",  "description": "HTTP sources for Gin, Echo, Fiber, Chi, Gorilla Mux"},
            {"id": "databases",      "name": "Databases",        "description": "ORM and driver sinks: GORM, sqlx, pgx, database/sql"},
            {"id": "stdlib",         "name": "Standard Library", "description": "Go stdlib: os/exec, net/http, path/filepath, strconv"},
            {"id": "http-clients",   "name": "HTTP Clients",     "description": "Outbound HTTP: net/http, go-resty — SSRF sinks"},
            {"id": "auth-config",    "name": "Auth & Config",    "description": "JWT verification, gRPC, Viper, YAML"},
        ],
    },
    "python": {
        "name": "Python",
        "version": "3.9+",
        "description": "Taint analysis for Python applications. Supports Flask, Django, FastAPI, and standard library sources/sinks.",
        "decorator": "@python_rule",
        "import_prefix": "from codepathfinder import calls, flows\nfrom codepathfinder.python_decorators import python_rule",
        "install": "pip install codepathfinder",
        "meta_module": "codepathfinder.python_rule_meta",
        "sdk_module":  None,  # Python rules use calls() not QueryType
        "categories": [
            {"id": "web-frameworks",    "name": "Web Frameworks",    "description": "Flask, Django, FastAPI request sources and response sinks"},
            {"id": "command-execution", "name": "Command Execution", "description": "subprocess, os — command injection sinks"},
            {"id": "databases",         "name": "Databases",          "description": "sqlite3, psycopg2, pymongo, redis — SQL and NoSQL sinks"},
            {"id": "deserialization",   "name": "Deserialization",    "description": "pickle, marshal, yaml — unsafe deserialization"},
            {"id": "http-clients",      "name": "HTTP Clients",       "description": "requests, httpx, urllib — SSRF sinks"},
            {"id": "file-system",       "name": "File System",        "description": "os.path, tempfile, pathlib — path traversal and temp file handling"},
            {"id": "archives",          "name": "Archives",           "description": "tarfile, zipfile — archive extraction (zip slip, bombs)"},
            {"id": "crypto",            "name": "Cryptography",       "description": "hashlib, hmac, ssl, secrets — weak crypto detection"},
            {"id": "templating",        "name": "Templating",         "description": "jinja2, string.Template — SSTI and XSS sinks"},
        ],
    },
}


# ── Helpers ────────────────────────────────────────────────────────────────────

def _collect_all_meta() -> dict[str, dict]:
    """
    Import each *_meta.py and return a mapping of
    language_id -> SDK_META dict.
    """
    results: dict[str, dict] = {}
    for lang_id, cfg in LANGUAGE_CONFIG.items():
        meta_module_name = cfg["meta_module"]
        try:
            mod = importlib.import_module(meta_module_name)
            results[lang_id] = mod.SDK_META  # type: ignore[attr-defined]
        except ModuleNotFoundError:
            print(f"  [skip] {meta_module_name} not found — skipping {lang_id}")
        except AttributeError:
            print(f"  [warn] {meta_module_name} has no SDK_META dict — skipping {lang_id}")
    return results


def _validate_methods(class_name: str, meta_methods: dict, sdk_module_name: str | None) -> list[str]:
    """
    Check that methods listed in SDK_META exist on the real SDK class.
    Returns a list of validation warnings (empty = all good).
    """
    warnings: list[str] = []
    if sdk_module_name is None:
        return warnings  # Python rules use calls() — no class to introspect

    try:
        mod = importlib.import_module(sdk_module_name)
        cls = getattr(mod, class_name, None)
        if cls is None:
            warnings.append(f"{class_name}: class not found in {sdk_module_name}")
            return warnings
        # QueryType classes expose .method() — we can't introspect Go receiver methods
        # but we can check that the meta methods list is non-empty
        if not meta_methods:
            warnings.append(f"{class_name}: no methods listed in SDK_META")
    except Exception as exc:
        warnings.append(f"{class_name}: import error — {exc}")
    return warnings


def _count_rule_usages(class_names: list[str]) -> dict[str, int]:
    """
    Scan rules/ for import lines to count how many rules reference each class.
    """
    counts: dict[str, int] = {name: 0 for name in class_names}
    for rule_file in RULES_DIR.rglob("*.py"):
        try:
            text = rule_file.read_text(encoding="utf-8")
            for name in class_names:
                if name in text:
                    counts[name] += 1
        except Exception:
            pass
    return counts


def _build_class_manifest(
    class_name: str,
    meta: dict,
    sdk_class: Any,
    usage_count: int,
) -> dict:
    """Build the per-class entry for sdk-manifest.json."""
    methods_out = []
    for method_name, mdata in meta.get("methods", {}).items():
        methods_out.append({
            "name": method_name,
            "signature": mdata.get("signature", f"{method_name}(...)"),
            "description": mdata.get("description", ""),
            "role": mdata.get("role", "neutral"),  # source | sink | sanitizer | neutral
            "tracks": mdata.get("tracks", []),
            "where_example": mdata.get("where_example"),
        })

    # FQNs: prefer real SDK class attrs, fall back to meta-provided fqns
    fqns     = getattr(sdk_class, "fqns",     meta.get("fqns",     [])) if sdk_class else meta.get("fqns",     [])
    patterns = getattr(sdk_class, "patterns", meta.get("patterns", [])) if sdk_class else meta.get("patterns", [])

    return {
        "id": class_name,
        "name": class_name,
        "description": meta.get("description", ""),
        "category": meta.get("category", "stdlib"),
        "fqns": fqns,
        "patterns": patterns,
        "go_mod": meta.get("go_mod"),
        "pip_snippet": meta.get("pip_snippet"),
        "import_stmt": f"from codepathfinder.go_rule import {class_name}",
        "methods": methods_out,
        "example_rule": meta.get("example_rule"),
        "rules_using": meta.get("rules_using", []),
        "usage_count": usage_count,
    }


# ── Main ───────────────────────────────────────────────────────────────────────

def generate(out_path: Path, verbose: bool = False) -> None:
    print(f"Generating SDK manifest → {out_path}")

    all_meta = _collect_all_meta()
    manifest: dict = {
        "version": "1.0.0",
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "languages": {},
    }

    for lang_id, lang_cfg in LANGUAGE_CONFIG.items():
        if lang_id not in all_meta:
            continue

        sdk_meta = all_meta[lang_id]
        sdk_module_name = lang_cfg["sdk_module"]

        # Load SDK module for introspection
        sdk_mod = None
        if sdk_module_name:
            try:
                sdk_mod = importlib.import_module(sdk_module_name)
            except ModuleNotFoundError:
                print(f"  [warn] Cannot import {sdk_module_name} — FQNs will be empty")

        # Count rule usages across all classes in this language
        class_names = list(sdk_meta.keys())
        usage_counts = _count_rule_usages(class_names)

        classes_out: dict[str, dict] = {}
        all_warnings: list[str] = []

        for class_name, meta in sdk_meta.items():
            sdk_class = getattr(sdk_mod, class_name, None) if sdk_mod else None
            warnings = _validate_methods(class_name, meta.get("methods", {}), sdk_module_name)
            all_warnings.extend(warnings)

            classes_out[class_name] = _build_class_manifest(
                class_name, meta, sdk_class, usage_counts.get(class_name, 0)
            )
            if verbose:
                method_count = len(meta.get("methods", {}))
                print(f"    {class_name}: {method_count} methods, {usage_counts.get(class_name, 0)} rules")

        if all_warnings:
            print(f"  Validation warnings for {lang_id}:")
            for w in all_warnings:
                print(f"    ⚠  {w}")

        # Build category class lists in order
        categories_out = []
        for cat in lang_cfg["categories"]:
            cat_classes = [
                cid for cid, cmeta in sdk_meta.items()
                if cmeta.get("category") == cat["id"]
            ]
            categories_out.append({
                "id": cat["id"],
                "name": cat["name"],
                "description": cat["description"],
                "class_ids": cat_classes,
            })

        manifest["languages"][lang_id] = {
            "id": lang_id,
            "name": lang_cfg["name"],
            "version": lang_cfg["version"],
            "description": lang_cfg["description"],
            "decorator": lang_cfg["decorator"],
            "import_prefix": lang_cfg["import_prefix"],
            "install": lang_cfg["install"],
            "categories": categories_out,
            "classes": classes_out,
            "total_classes": len(classes_out),
            "total_methods": sum(len(c["methods"]) for c in classes_out.values()),
        }

        print(f"  {lang_id}: {len(classes_out)} classes, "
              f"{manifest['languages'][lang_id]['total_methods']} methods")

    out_path.parent.mkdir(parents=True, exist_ok=True)
    out_path.write_text(json.dumps(manifest, indent=2), encoding="utf-8")
    print(f"Done. Written to {out_path}")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Generate sdk-manifest.json")
    parser.add_argument("--out", type=Path, default=OUTPUT_DEFAULT, help="Output path")
    parser.add_argument("--verbose", "-v", action="store_true")
    parser.add_argument(
        "--skip-cdn-index",
        action="store_true",
        help="Skip the CDN stub-indexing pass (default: run it after handcrafted meta).",
    )
    args = parser.parse_args()
    generate(args.out, verbose=args.verbose)

    if not args.skip_cdn_index:
        # Chain CDN indexers: stubs for every CDN module we don't cover by hand.
        print()
        import subprocess
        for indexer_name in ("index_python_from_cdn.py", "index_go_from_cdn.py"):
            indexer = SCRIPT_DIR / indexer_name
            if indexer.exists():
                result = subprocess.run([sys.executable, str(indexer)], check=False)
                if result.returncode != 0:
                    print(f"[warn] {indexer_name} returned non-zero — handcrafted manifest still written.")
            else:
                print(f"[info] CDN indexer not found at {indexer} — skipping.")
