#!/usr/bin/env python3
"""
index_python_from_cdn.py — Merge CDN Python registries into sdk-manifest.json.

This script runs AFTER generate_sdk_manifest.py. It reads the website manifest,
pulls stdlib + third-party module lists from the CDN, and adds a stub entry for
every module that the handcrafted python_rule_meta.py doesn't already cover.

Goal: complete discoverability. Rule writers and code-quality reviewers should
be able to search the SDK reference for any module and find at least a
canonical FQN and category, even if we haven't annotated source/sink roles yet.

Usage:
    python scripts/index_python_from_cdn.py
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
    "https://assets.codepathfinder.dev/registries/python3.11/stdlib/v1/manifest.json"
)
THIRDPARTY_MANIFEST_URL = (
    "https://assets.codepathfinder.dev/registries/thirdparty/v1/manifest.json"
)

# Category rules applied in order. First match wins.
# Modules not matching any rule land in "utilities".
CATEGORY_EXACT: dict[str, str] = {
    # command-execution
    "subprocess": "command-execution",
    "os": "command-execution",
    "pty": "command-execution",
    "pexpect": "command-execution",
    "docker": "command-execution",
    "ctypes": "command-execution",
    "cffi": "command-execution",
    "fcntl": "command-execution",
    "signal": "command-execution",
    # deserialization
    "pickle": "deserialization",
    "pickletools": "deserialization",
    "marshal": "deserialization",
    "json": "deserialization",
    "simplejson": "deserialization",
    "yaml": "deserialization",
    "csv": "deserialization",
    "xml": "deserialization",
    "lxml": "deserialization",
    "defusedxml": "deserialization",
    "xmltodict": "deserialization",
    "xmlrpc": "deserialization",
    "shelve": "deserialization",
    "dbm": "deserialization",
    "pyasn1": "deserialization",
    "toml": "deserialization",
    "tomllib": "deserialization",
    "plistlib": "deserialization",
    "ast": "deserialization",
    "pyexpat": "deserialization",
    # databases
    "sqlite3": "databases",
    "psycopg2": "databases",
    "pymongo": "databases",
    "redis": "databases",
    "sqlalchemy": "databases",
    "pymysql": "databases",
    "MySQLdb": "databases",
    "ldap3": "databases",
    "hdbcli": "databases",
    "ibm_db": "databases",
    "playhouse": "databases",
    "pony": "databases",
    # http-clients
    "requests": "http-clients",
    "urllib": "http-clients",
    "http": "http-clients",
    "socket": "http-clients",
    "ftplib": "http-clients",
    "telnetlib": "http-clients",
    "smtplib": "http-clients",
    "smtpd": "http-clients",
    "httpx": "http-clients",
    "httplib2": "http-clients",
    "aiohttp": "http-clients",
    "pycurl": "http-clients",
    "imaplib": "http-clients",
    "poplib": "http-clients",
    "nntplib": "http-clients",
    "email": "http-clients",
    "ipaddress": "http-clients",
    "netaddr": "http-clients",
    "netifaces": "http-clients",
    "boto3": "http-clients",
    "aws_xray_sdk": "http-clients",
    "mailbox": "http-clients",
    "mailcap": "http-clients",
    "netrc": "http-clients",
    "webob": "http-clients",
    "slumber": "http-clients",
    "pika": "http-clients",
    "pysocks": "http-clients",
    # file-system
    "pathlib": "file-system",
    "tempfile": "file-system",
    "shutil": "file-system",
    "logging": "file-system",
    "aiofiles": "file-system",
    "glob": "file-system",
    "fnmatch": "file-system",
    "re": "file-system",
    "regex": "file-system",
    "configparser": "file-system",
    "dockerfile_parse": "file-system",
    "mimetypes": "file-system",
    "fileinput": "file-system",
    # archives
    "tarfile": "archives",
    "zipfile": "archives",
    "zlib": "archives",
    "gzip": "archives",
    "bz2": "archives",
    "lzma": "archives",
    "zipapp": "archives",
    "zipimport": "archives",
    "zstd": "archives",
    # crypto
    "hashlib": "crypto",
    "hmac": "crypto",
    "secrets": "crypto",
    "random": "crypto",
    "ssl": "crypto",
    "jose": "crypto",
    "authlib": "crypto",
    "paramiko": "crypto",
    "getpass": "crypto",
    "crypt": "crypto",
    "pysftp": "crypto",
    "jwt": "crypto",
    "jwcrypto": "crypto",
    "jks": "crypto",
    "oauthlib": "crypto",
    "requests_oauthlib": "crypto",
    "auth0": "crypto",
    "hvac": "crypto",
    "passpy": "crypto",
    "tgcrypto": "crypto",
    "zxcvbn": "crypto",
    "cryptography": "crypto",
    # templating
    "jinja2": "templating",
    "bleach": "templating",
    "html": "templating",
    "html5lib": "templating",
    "string": "templating",
    "chevron": "templating",
    "markdown": "templating",
    "docutils": "templating",
    "reportlab": "templating",
    "fpdf": "templating",
    "webencodings": "templating",
    # web-frameworks
    "flask": "web-frameworks",
    "flask_cors": "web-frameworks",
    "flask_migrate": "web-frameworks",
    "flask_socketio": "web-frameworks",
    "django": "web-frameworks",
    "django_filters": "web-frameworks",
    "fastapi": "web-frameworks",
    "starlette": "web-frameworks",
    "rest_framework": "web-frameworks",
    "channels": "web-frameworks",
    "gunicorn": "web-frameworks",
    "waitress": "web-frameworks",
    "uwsgi": "web-frameworks",
    "cgi": "web-frameworks",
    "cgitb": "web-frameworks",
    "wsgiref": "web-frameworks",
    "pydantic": "web-frameworks",
    "wtforms": "web-frameworks",
    "jsonschema": "web-frameworks",
    "celery": "web-frameworks",
    "grpc": "web-frameworks",
    "grpc_channelz": "web-frameworks",
    "grpc_health": "web-frameworks",
    "grpc_reflection": "web-frameworks",
    "grpc_status": "web-frameworks",
    "simple_websocket": "web-frameworks",
    "gevent": "web-frameworks",
    "greenlet": "web-frameworks",
    "fanstatic": "web-frameworks",
    "werkzeug": "web-frameworks",
}

CATEGORY_PATTERNS: list[tuple[str, list[str]]] = [
    (
        "language",
        [
            "abc",
            "typing",
            "types",
            "dataclasses",
            "enum",
            "collections",
            "contextlib",
            "contextvars",
            "functools",
            "itertools",
            "operator",
            "numbers",
            "decimal",
            "fractions",
            "statistics",
            "math",
            "cmath",
            "array",
            "bisect",
            "heapq",
            "graphlib",
            "weakref",
            "copy",
            "copyreg",
            "gc",
            "errno",
            "atexit",
            "warnings",
            "traceback",
            "sys",
            "platform",
            "sysconfig",
            "builtins",
            "importlib",
            "imp",
            "runpy",
            "site",
            "compileall",
            "keyword",
            "symtable",
            "token",
            "tokenize",
            "linecache",
            "pkgutil",
            "pyclbr",
            "pydoc",
            "pydoc_data",
            "rlcompleter",
            "code",
            "codeop",
            "dis",
            "inspect",
            "opcode",
            "py_compile",
            "tabnanny",
            "encodings",
            "codecs",
            "textwrap",
            "unicodedata",
            "stringprep",
            "readline",
            "modulefinder",
            "struct",
            "faulthandler",
            "resource",
            "select",
            "selectors",
            "termios",
            "singledispatch",
            "six",
            "decorator",
            "deprecated",
            "mypy_extensions",
            "entrypoints",
            "toposort",
            "boltons",
            "cachetools",
            "objgraph",
            "retry",
            "punq",
            "watchpoints",
            "usersettings",
        ],
    ),
    (
        "concurrency",
        [
            "asyncio",
            "asynchat",
            "asyncore",
            "threading",
            "multiprocessing",
            "queue",
            "sched",
            "concurrent",
        ],
    ),
    (
        "testing",
        [
            "unittest",
            "doctest",
            "atheris",
            "mock",
            "assertpy",
            "behave",
            "editdistance",
            "pytest_lazy_fixture",
        ],
    ),
    (
        "datetime",
        [
            "datetime",
            "time",
            "zoneinfo",
            "locale",
            "gettext",
            "calendar",
            "dateparser_data",
            "dateutil",
            "croniter",
            "pytz",
            "python_crontab",
            "lunardate",
            "convertdate",
            "ephem",
            "icalendar",
            "workalendar",
            "pymeeus",
            "pyluach",
            "tzdata",
            "vobject",
            "rfc3339_validator",
        ],
    ),
    (
        "io",
        [
            "io",
            "filecmp",
            "difflib",
            "stat",
            "pipes",
            "ossaudiodev",
            "wave",
            "sunau",
            "aifc",
            "audioop",
            "chunk",
            "sndhdr",
            "imghdr",
            "uu",
            "quopri",
            "base64",
            "binascii",
            "xdrlib",
            "mmap",
            "datauri",
            "dirhash",
            "binaryornot",
            "str2bool",
            "unidiff",
            "whatthepatch",
            "xmldiff",
            "untangle",
            "olefile",
        ],
    ),
    (
        "cli",
        [
            "argparse",
            "getopt",
            "optparse",
            "cmd",
            "shlex",
            "click",
            "colorama",
            "colorful",
            "consolemenu",
            "keyboard",
            "pyautogui",
            "pyperclip",
            "pynput",
            "pyscreeze",
            "send2trash",
            "tabulate",
            "tqdm",
            "pygments",
            "capturer",
            "wurlitzer",
        ],
    ),
    (
        "gui",
        [
            "tkinter",
            "curses",
            "turtle",
            "turtledemo",
            "pyi_splash",
            "ttkthemes",
            "Xlib",
            "jack",
            "pyaudio",
        ],
    ),
    (
        "data-science",
        [
            "tensorflow",
            "networkx",
            "geopandas",
            "shapely",
            "seaborn",
            "pycocotools",
            "hnswlib",
            "resampy",
            "openpyxl",
            "et_xmlfile",
            "xlrd",
            "pyfarmhash",
            "qrcode",
            "qrbill",
        ],
    ),
    (
        "dev-tools",
        [
            "distutils",
            "ensurepip",
            "venv",
            "lib2to3",
            "idlelib",
            "bdb",
            "pdb",
            "trace",
            "tracemalloc",
            "timeit",
            "cProfile",
            "profile",
            "pstats",
            "flake8",
            "pyflakes",
            "pep8_naming",
            "gdb",
            "pythonwin",
            "portpicker",
            "dockerfile_parse",
            "jenkins",
            "jmespath",
            "fire",
        ],
    ),
]


def fetch_json(url: str) -> dict:
    req = urllib.request.Request(
        url, headers={"User-Agent": "codepathfinder-indexer/1.0"}
    )
    with urllib.request.urlopen(req, timeout=30) as resp:
        data: dict = json.loads(resp.read().decode("utf-8"))
        return data


def to_pascal_case(name: str) -> str:
    """abc -> PyAbc; xml.etree -> PyXmlEtree; http.client -> PyHttpClient."""
    parts = re.split(r"[._-]+", name)
    return "Py" + "".join(
        p[:1].upper() + p[1:].lower() if p else "" for p in parts if p
    )


def categorize(module_name: str) -> str:
    # Try exact first, then prefix matching
    if module_name in CATEGORY_EXACT:
        return CATEGORY_EXACT[module_name]
    head = module_name.split(".", 1)[0]
    if head in CATEGORY_EXACT:
        return CATEGORY_EXACT[head]
    for category, names in CATEGORY_PATTERNS:
        if module_name in names or head in names:
            return category
    return "utilities"


def extract_top_methods(detail: dict, limit: int = 10) -> list[dict]:
    """Pull a handful of public functions/classes from a per-module CDN detail JSON."""
    methods = []

    # Module-level functions
    for fn_name, fn in (detail.get("functions") or {}).items():
        if fn_name.startswith("_"):
            continue
        params = fn.get("parameters") or fn.get("params") or []
        param_strs: list[str] = []
        for p in params:
            if isinstance(p, dict):
                n = p.get("name", "")
                if n:
                    param_strs.append(n)
            elif isinstance(p, str):
                param_strs.append(p)
        sig = f"{fn_name}({', '.join(param_strs)})"
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

    # Classes (flatten to class name only)
    for cls_name, cls in (detail.get("classes") or {}).items():
        if cls_name.startswith("_"):
            continue
        doc = (cls.get("docstring") or cls.get("description") or "").strip().split(
            "\n"
        )[0][:180] or f"{cls_name} class."
        methods.append(
            {
                "name": cls_name,
                "signature": f"{cls_name}(...)",
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


def build_stub(module: dict, source: str, base_url: str, install_hint: str) -> dict:
    name = module["name"]
    filename = module["file"]
    detail = try_fetch_detail(base_url, filename) or {}
    methods = extract_top_methods(detail, limit=10)
    return {
        "id": to_pascal_case(name),
        "name": to_pascal_case(name),
        "description": (
            f"{source} module — {name}. Auto-indexed from CDN. "
            "Method-level security roles have not been annotated; rule writers should inspect the source before use."
        ),
        "category": categorize(name),
        "fqns": [name],
        "patterns": [],
        "go_mod": None,
        "pip_snippet": install_hint,
        "import_stmt": (
            f"import {name}"
            if "." not in name
            else f"from {name.rsplit('.', 1)[0]} import {name.rsplit('.', 1)[1]}"
        ),
        "methods": methods,
        "example_rule": None,
        "rules_using": [],
        "usage_count": 0,
    }


EXTRA_CATEGORIES = [
    {
        "id": "language",
        "name": "Language Features",
        "description": "Python language primitives: typing, collections, abc, functools",
    },
    {
        "id": "concurrency",
        "name": "Concurrency",
        "description": "asyncio, threading, multiprocessing, queue",
    },
    {
        "id": "datetime",
        "name": "Date & Time",
        "description": "datetime, time, zoneinfo, dateutil",
    },
    {
        "id": "io",
        "name": "I/O & Encoding",
        "description": "io streams, base64, binascii, plistlib",
    },
    {
        "id": "cli",
        "name": "CLI & Terminal",
        "description": "argparse, click, colorama, tqdm",
    },
    {"id": "gui", "name": "GUI", "description": "tkinter, curses, turtle"},
    {
        "id": "testing",
        "name": "Testing",
        "description": "unittest, doctest, mock, pytest tooling",
    },
    {
        "id": "dev-tools",
        "name": "Developer Tools",
        "description": "distutils, venv, pdb, linters",
    },
    {
        "id": "data-science",
        "name": "Data Science",
        "description": "tensorflow, networkx, geopandas, openpyxl",
    },
    {
        "id": "utilities",
        "name": "Utilities",
        "description": "Miscellaneous modules without a dedicated category",
    },
]


def main() -> None:
    if not MANIFEST_PATH.exists():
        raise SystemExit(
            f"Manifest not found at {MANIFEST_PATH}. Run generate_sdk_manifest.py first."
        )

    manifest = json.loads(MANIFEST_PATH.read_text())
    py = manifest["languages"].setdefault("python", {})
    classes: dict[str, Any] = py.setdefault("classes", {})
    categories = py.setdefault("categories", [])

    # Reverse lookup: module name -> class id (if already handcrafted)
    covered_fqns: set[str] = set()
    for cid, centry in classes.items():
        for fq in centry.get("fqns", []) or []:
            covered_fqns.add(fq.split(".")[0])
            covered_fqns.add(fq)

    # Add new categories if missing
    existing_cat_ids = {c["id"] for c in categories}
    for extra in EXTRA_CATEGORIES:
        if extra["id"] not in existing_cat_ids:
            categories.append({**extra, "class_ids": []})

    # Fetch CDN manifests
    print("Fetching CDN manifests...")
    stdlib = fetch_json(STDLIB_MANIFEST_URL)
    thirdparty = fetch_json(THIRDPARTY_MANIFEST_URL)

    stdlib_base = stdlib.get("base_url", STDLIB_MANIFEST_URL.rsplit("/", 1)[0])
    thirdparty_base = thirdparty.get(
        "base_url", THIRDPARTY_MANIFEST_URL.rsplit("/", 1)[0]
    )

    added = 0
    skipped = 0

    for module in stdlib.get("modules", []):
        name = module["name"]
        if name in covered_fqns:
            skipped += 1
            continue
        stub = build_stub(
            module, "Python stdlib", stdlib_base, "# stdlib — no install required"
        )
        classes[stub["id"]] = stub
        added += 1

    for module in thirdparty.get("modules", []):
        name = module["name"]
        if name in covered_fqns:
            skipped += 1
            continue
        stub = build_stub(
            module, "Third-party Python package", thirdparty_base, f"pip install {name}"
        )
        classes[stub["id"]] = stub
        added += 1

    # Rebuild category class_ids lists
    id_by_cat: dict[str, list[str]] = {c["id"]: [] for c in categories}
    for cid, centry in classes.items():
        cat = centry.get("category", "utilities")
        if cat not in id_by_cat:
            id_by_cat[cat] = []
            categories.append(
                {"id": cat, "name": cat.title(), "description": "", "class_ids": []}
            )
        id_by_cat[cat].append(cid)

    for cat in categories:
        cat["class_ids"] = sorted(id_by_cat.get(cat["id"], []))

    py["total_classes"] = len(classes)
    py["total_methods"] = sum(len(c["methods"]) for c in classes.values())

    MANIFEST_PATH.write_text(json.dumps(manifest, indent=2))
    print(f"Added {added} stubs, skipped {skipped} already-covered modules.")
    print(f"Python total: {py['total_classes']} classes, {py['total_methods']} methods")


if __name__ == "__main__":
    main()
