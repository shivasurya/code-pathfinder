#!/usr/bin/env python3
"""
index_go_from_cdn.py — Merge Go stdlib + third-party registries from CDN into sdk-manifest.json.

Runs AFTER generate_sdk_manifest.py. For each Go package on the CDN that isn't already
covered by a handcrafted go_rule_meta.py entry, generate a stub so rule writers and
code-quality reviewers can still discover it.

Usage:
    python scripts/index_go_from_cdn.py
"""

from __future__ import annotations

import json
import re
import sys
import urllib.request
from pathlib import Path
from typing import Any

SCRIPT_DIR = Path(__file__).parent
MANIFEST_PATH = (
    SCRIPT_DIR.parent.parent.parent / "cpf-website" / "public" / "sdk-manifest.json"
)

STDLIB_MANIFEST_URL = (
    "https://assets.codepathfinder.dev/registries/go1.21/stdlib/v1/manifest.json"
)
THIRDPARTY_MANIFEST_URL = (
    "https://assets.codepathfinder.dev/registries/go-thirdparty/v1/manifest.json"
)

# Category rules applied in order. Head of the import path is checked first.
CATEGORY_EXACT: dict[str, str] = {
    # stdlib security-relevant (already covered by handcrafted entries; these are fallbacks)
    "os": "stdlib",
    "os/exec": "stdlib",
    "net/http": "stdlib",
    "net": "stdlib",
    "net/url": "stdlib",
    "net/smtp": "stdlib",
    "crypto": "stdlib",
    "crypto/tls": "stdlib",
    "crypto/x509": "stdlib",
    "crypto/aes": "stdlib",
    "crypto/cipher": "stdlib",
    "crypto/hmac": "stdlib",
    "crypto/rand": "stdlib",
    "crypto/sha256": "stdlib",
    "crypto/sha512": "stdlib",
    "encoding/json": "stdlib",
    "encoding/xml": "stdlib",
    "encoding/base64": "stdlib",
    "encoding/binary": "stdlib",
    "encoding/csv": "stdlib",
    "encoding/gob": "stdlib",
    "encoding/hex": "stdlib",
    "database/sql": "stdlib",
    "path/filepath": "stdlib",
    "io": "stdlib",
    "io/fs": "stdlib",
    "bufio": "stdlib",
    "archive/tar": "stdlib",
    "archive/zip": "stdlib",
    "html/template": "stdlib",
    "text/template": "stdlib",
    "mime/multipart": "stdlib",
    "log": "stdlib",
    "log/slog": "stdlib",
    "math/rand": "stdlib",
    "plugin": "stdlib",
    "reflect": "stdlib",
    "regexp": "stdlib",
    "runtime": "stdlib",
    "strconv": "stdlib",
    "strings": "stdlib",
    "sync": "stdlib",
    "syscall": "stdlib",
    "time": "stdlib",
    "fmt": "stdlib",
    "context": "stdlib",
}

# Prefix-based routing for third-party packages
THIRD_PARTY_PREFIXES: list[tuple[str, str]] = [
    # Web frameworks
    ("github.com/gin-gonic/gin", "web-frameworks"),
    ("github.com/labstack/echo", "web-frameworks"),
    ("github.com/gofiber/fiber", "web-frameworks"),
    ("github.com/go-chi/chi", "web-frameworks"),
    ("github.com/gorilla/mux", "web-frameworks"),
    ("github.com/flosch/pongo2", "web-frameworks"),
    ("github.com/go-playground/validator", "web-frameworks"),
    # Databases
    ("gorm.io/gorm", "databases"),
    ("github.com/jackc/pgx", "databases"),
    ("github.com/jmoiron/sqlx", "databases"),
    ("github.com/redis/go-redis", "databases"),
    ("go.mongodb.org/mongo-driver", "databases"),
    ("k8s.io/client-go", "databases"),
    # HTTP clients
    ("github.com/go-resty/resty", "http-clients"),
    ("github.com/aws/aws-sdk-go-v2", "http-clients"),
    ("cloud.google.com/go", "http-clients"),
    # Auth / Config
    ("github.com/golang-jwt/jwt", "auth-config"),
    ("github.com/spf13/viper", "auth-config"),
    ("github.com/spf13/afero", "auth-config"),
    ("gopkg.in/yaml.v3", "auth-config"),
    ("github.com/pelletier/go-toml", "auth-config"),
    ("google.golang.org/grpc", "auth-config"),
    # Logging / observability
    ("github.com/sirupsen/logrus", "auth-config"),
    ("go.uber.org/zap", "auth-config"),
    # Testing
    ("github.com/stretchr/testify", "auth-config"),
    # CLI
    ("github.com/codeskyblue/go-sh", "auth-config"),
]


def fetch_json(url: str) -> dict:
    req = urllib.request.Request(
        url, headers={"User-Agent": "codepathfinder-indexer/1.0"}
    )
    with urllib.request.urlopen(req, timeout=30) as resp:
        data: dict = json.loads(resp.read().decode("utf-8"))
        return data


def to_pascal_case_go(import_path: str) -> str:
    """Convert Go import path to a PascalCase class id.

    github.com/sirupsen/logrus         -> GoLogrus
    github.com/aws/aws-sdk-go-v2       -> GoAwsSdkGoV2
    cloud.google.com/go                -> GoCloudGoogle
    crypto/sha256                      -> GoCryptoSha256
    net/http                           -> GoNetHttp
    """
    # Strip common prefixes
    segs = import_path.replace("github.com/", "").replace("gopkg.in/", "").split("/")
    segs = [s for s in segs if s and s not in ("go",)]  # drop trailing /go segment
    if not segs:
        segs = import_path.split("/")

    out_parts: list[str] = []
    for seg in segs:
        # Remove version suffixes like /v2, /v5 at end of path segment
        if re.fullmatch(r"v\d+", seg):
            continue
        # Split by . or -
        sub = re.split(r"[.\-]+", seg)
        for s in sub:
            if s and not re.fullmatch(r"v\d+", s):
                out_parts.append(s[:1].upper() + s[1:].lower())

    if not out_parts:
        return "GoUnknown"
    return "Go" + "".join(out_parts)


def categorize_stdlib(import_path: str) -> str:
    """Go stdlib all land in the 'stdlib' category by definition."""
    return "stdlib"


def categorize_thirdparty(import_path: str) -> str:
    for prefix, cat in THIRD_PARTY_PREFIXES:
        if import_path == prefix or import_path.startswith(prefix + "/"):
            return cat
    # Default third-party bucket
    return "auth-config"


def extract_top_symbols(detail: dict, limit: int = 10) -> list[dict]:
    """Pull top functions, types, and methods from a Go per-package CDN detail JSON."""
    methods: list[dict] = []

    # Functions
    for fn_name, fn in (detail.get("functions") or {}).items():
        if fn_name.startswith("_") or fn_name[:1].islower():
            continue  # unexported
        params = fn.get("parameters") or fn.get("params") or []
        param_strs: list[str] = []
        for p in params:
            if isinstance(p, dict):
                n = p.get("name") or ""
                t = p.get("type") or p.get("type_name") or ""
                param_strs.append(f"{n} {t}".strip() if n and t else (n or t or ""))
            elif isinstance(p, str):
                param_strs.append(p)
        returns = fn.get("returns") or fn.get("return_types") or []
        ret_str = ""
        if returns:
            ret_types: list[str] = []
            for r in returns:
                if isinstance(r, dict):
                    ret_types.append(r.get("type") or r.get("type_name") or "")
                elif isinstance(r, str):
                    ret_types.append(r)
            if len(ret_types) == 1:
                ret_str = " " + ret_types[0]
            elif ret_types:
                ret_str = " (" + ", ".join(t for t in ret_types if t) + ")"
        sig = f"{fn_name}({', '.join(param_strs)}){ret_str}".strip()
        doc = (fn.get("docstring") or fn.get("description") or "").strip().split("\n")[
            0
        ][:180] or f"{fn_name} function."
        methods.append(
            {
                "name": fn_name,
                "signature": sig,
                "description": doc,
                "role": "neutral",
                "tracks": [],
            }
        )
        if len(methods) >= limit:
            return methods

    # Types (structs / interfaces)
    for type_name, tdef in (detail.get("types") or {}).items():
        if type_name.startswith("_") or type_name[:1].islower():
            continue
        doc = (tdef.get("docstring") or tdef.get("description") or "").strip().split(
            "\n"
        )[0][:180] or f"{type_name} type."
        methods.append(
            {
                "name": type_name,
                "signature": f"type {type_name} ...",
                "description": doc,
                "role": "neutral",
                "tracks": [],
            }
        )
        if len(methods) >= limit:
            return methods

    return methods


def try_fetch_detail(base_url: str, filename: str) -> dict | None:
    try:
        return fetch_json(f"{base_url}/{filename}")
    except Exception as e:
        print(f"  [skip] {filename}: {e}", file=sys.stderr)
        return None


def build_stub(
    *,
    import_path: str,
    source: str,
    base_url: str,
    file_hint: str,
    go_mod: str | None,
    category: str,
) -> dict:
    detail = try_fetch_detail(base_url, file_hint) or {}
    methods = extract_top_symbols(detail, limit=10)
    class_id = to_pascal_case_go(import_path)
    return {
        "id": class_id,
        "name": class_id,
        "description": (
            f"{source} package — {import_path}. Auto-indexed from CDN. "
            "Method-level security roles have not been annotated; rule writers should inspect the source before use."
        ),
        "category": category,
        "fqns": [import_path],
        "patterns": [],
        "go_mod": go_mod,
        "pip_snippet": None,
        "import_stmt": f"from codepathfinder.go_rule import ...  # {import_path}",
        "methods": methods,
        "example_rule": None,
        "rules_using": [],
        "usage_count": 0,
    }


def main() -> None:
    if not MANIFEST_PATH.exists():
        raise SystemExit(
            f"Manifest not found at {MANIFEST_PATH}. Run generate_sdk_manifest.py first."
        )

    manifest = json.loads(MANIFEST_PATH.read_text())
    go = manifest["languages"].setdefault("golang", {})
    classes: dict[str, Any] = go.setdefault("classes", {})
    categories = go.setdefault("categories", [])

    # Build set of covered import paths from existing fqns
    covered_paths: set[str] = set()
    covered_ids: set[str] = set(classes.keys())
    for cid, centry in classes.items():
        for fqn in centry.get("fqns", []) or []:
            # strip .TypeName suffix where present
            if "/" in fqn:
                last = fqn.split("/")[-1]
                if "." in last:
                    path = fqn.rsplit(".", 1)[0]
                    covered_paths.add(path)
                    covered_paths.add(fqn)
                else:
                    covered_paths.add(fqn)
            else:
                covered_paths.add(fqn)
                if "." in fqn:
                    covered_paths.add(fqn.split(".", 1)[0])

    # Fetch CDN manifests
    print("Fetching Go CDN manifests...")
    stdlib = fetch_json(STDLIB_MANIFEST_URL)
    thirdparty = fetch_json(THIRDPARTY_MANIFEST_URL)

    stdlib_base = stdlib.get("base_url", STDLIB_MANIFEST_URL.rsplit("/", 1)[0])
    thirdparty_base = thirdparty.get(
        "base_url", THIRDPARTY_MANIFEST_URL.rsplit("/", 1)[0]
    )

    added = 0
    skipped = 0
    collisions = 0

    # stdlib
    for pkg in stdlib.get("packages", []):
        import_path = pkg["import_path"]
        if import_path in covered_paths:
            skipped += 1
            continue
        cat = "stdlib"
        class_id = to_pascal_case_go(import_path)
        if class_id in covered_ids:
            # Rare collision (different import paths producing the same class id); disambiguate
            class_id = class_id + "Stdlib"
            collisions += 1
        stub = build_stub(
            import_path=import_path,
            source="Go stdlib",
            base_url=stdlib_base,
            file_hint=f"{import_path.replace('/', '_')}_stdlib.json",
            go_mod="// standard library — no go.mod entry required",
            category=cat,
        )
        stub["id"] = class_id
        stub["name"] = class_id
        classes[class_id] = stub
        covered_ids.add(class_id)
        added += 1

    # third-party
    for pkg in thirdparty.get("packages", []):
        import_path = pkg["import_path"]
        if import_path in covered_paths:
            skipped += 1
            continue
        cat = categorize_thirdparty(import_path)
        class_id = to_pascal_case_go(import_path)
        if class_id in covered_ids:
            class_id = class_id + "Pkg"
            collisions += 1
        # For third-party, file hint uses different encoding per CDN docs
        file_hint = import_path.replace("/", "_") + ".json"
        stub = build_stub(
            import_path=import_path,
            source="Go third-party",
            base_url=thirdparty_base,
            file_hint=file_hint,
            go_mod=f"require {import_path} latest",
            category=cat,
        )
        stub["id"] = class_id
        stub["name"] = class_id
        classes[class_id] = stub
        covered_ids.add(class_id)
        added += 1

    # Rebuild category class_ids lists from scratch
    id_by_cat: dict[str, list[str]] = {c["id"]: [] for c in categories}
    for cid, centry in classes.items():
        cat = centry.get("category", "stdlib")
        if cat not in id_by_cat:
            # Unknown category encountered; register it
            categories.append(
                {
                    "id": cat,
                    "name": cat.replace("-", " ").title(),
                    "description": "",
                    "class_ids": [],
                }
            )
            id_by_cat[cat] = []
        id_by_cat[cat].append(cid)

    for cat in categories:
        cat["class_ids"] = sorted(id_by_cat.get(cat["id"], []))

    go["total_classes"] = len(classes)
    go["total_methods"] = sum(len(c["methods"]) for c in classes.values())

    MANIFEST_PATH.write_text(json.dumps(manifest, indent=2))
    print(
        f"Added {added} Go stubs, skipped {skipped} already-covered packages, resolved {collisions} id collisions."
    )
    print(f"Go total: {go['total_classes']} classes, {go['total_methods']} methods")


if __name__ == "__main__":
    main()
