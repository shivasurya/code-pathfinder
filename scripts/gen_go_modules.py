#!/usr/bin/env python3
"""Generate go.mod + go.sum for every Go rule's positive and negative test directories.

Each directory becomes an isolated Go module so it can be treated as a real project
by the Code Pathfinder engine (and displayed correctly in the cpf-website playground).
"""

import os
import subprocess
import sys
from pathlib import Path

RULES_DIR = Path(__file__).parent.parent / "rules" / "golang"
GO_VERSION = "1.21"

# Dependency map: rule_id → {pos: [...], neg: [...]}
# Values are module paths as they appear in import statements (without version).
# go mod tidy resolves the latest compatible version automatically.
DEPS: dict[str, dict[str, list[str]]] = {
    # ── Crypto (stdlib only) ─────────────────────────────────────────────────
    "GO-CRYPTO-001": {"pos": [], "neg": []},
    "GO-CRYPTO-002": {"pos": [], "neg": []},
    "GO-CRYPTO-003": {"pos": [], "neg": []},
    "GO-CRYPTO-004": {"pos": [], "neg": []},
    "GO-CRYPTO-005": {"pos": [], "neg": []},
    # ── SQL injection via GORM ────────────────────────────────────────────────
    "GO-GORM-SQLI-001": {
        "pos": ["github.com/gin-gonic/gin", "gorm.io/gorm"],
        "neg": ["gorm.io/gorm"],
    },
    "GO-GORM-SQLI-002": {
        "pos": ["github.com/gin-gonic/gin", "gorm.io/gorm"],
        "neg": ["gorm.io/gorm"],
    },
    # ── JWT ──────────────────────────────────────────────────────────────────
    "GO-JWT-002": {
        "pos": ["github.com/golang-jwt/jwt/v5"],
        "neg": ["github.com/golang-jwt/jwt/v5"],
    },
    # ── Network ──────────────────────────────────────────────────────────────
    "GO-NET-001": {"pos": [], "neg": []},
    "GO-NET-002": {
        "pos": ["google.golang.org/grpc"],
        "neg": ["google.golang.org/grpc"],
    },
    # ── Path traversal ───────────────────────────────────────────────────────
    "GO-PATH-001": {
        "pos": ["github.com/gin-gonic/gin"],
        "neg": [],
    },
    # ── Open redirect ────────────────────────────────────────────────────────
    "GO-REDIRECT-001": {
        "pos": ["github.com/gin-gonic/gin", "github.com/labstack/echo/v4"],
        "neg": [],
    },
    # ── SQL injection (raw db) ────────────────────────────────────────────────
    "GO-SEC-001": {
        "pos": ["github.com/gin-gonic/gin"],
        "neg": [],
    },
    # ── Command injection ────────────────────────────────────────────────────
    "GO-SEC-002": {
        "pos": ["github.com/gin-gonic/gin"],
        "neg": [],
    },
    # ── Hardcoded credentials ────────────────────────────────────────────────
    "GO-SEC-004": {"pos": [], "neg": []},
    # ── SQL injection via pgx ─────────────────────────────────────────────────
    "GO-SQLI-002": {
        "pos": ["github.com/gin-gonic/gin", "github.com/jackc/pgx/v5"],
        "neg": ["github.com/jackc/pgx/v5"],
    },
    # ── SQL injection via sqlx ────────────────────────────────────────────────
    "GO-SQLI-003": {
        "pos": ["github.com/gin-gonic/gin", "github.com/jmoiron/sqlx"],
        "neg": ["github.com/jmoiron/sqlx"],
    },
    # ── SSRF ─────────────────────────────────────────────────────────────────
    "GO-SSRF-001": {
        "pos": ["github.com/gin-gonic/gin"],
        "neg": [],
    },
    "GO-SSRF-002": {
        "pos": ["github.com/gin-gonic/gin"],
        "neg": [],
    },
    # ── XSS ──────────────────────────────────────────────────────────────────
    "GO-XSS-001": {"pos": [], "neg": []},
    "GO-XSS-002": {"pos": [], "neg": []},
    "GO-XSS-003": {"pos": [], "neg": []},
}


def write_go_mod(directory: Path, rule_id: str, variant: str, deps: list[str]) -> None:
    """Write a minimal go.mod — no require block; go mod tidy resolves versions."""
    module_name = f"example.com/{rule_id.lower()}/{variant}"
    go_mod = f"module {module_name}\n\ngo {GO_VERSION}\n"
    (directory / "go.mod").write_text(go_mod)
    print(f"  wrote go.mod → {directory.relative_to(RULES_DIR.parent.parent)}")


def run_go_mod_tidy(directory: Path) -> bool:
    """Run 'go mod tidy' and return True on success."""
    result = subprocess.run(
        ["go", "mod", "tidy"],
        cwd=directory,
        capture_output=True,
        text=True,
        timeout=120,
    )
    if result.returncode != 0:
        print(f"  ERROR in {directory}:\n{result.stderr}", file=sys.stderr)
        return False
    return True


def process_rule(rule_id: str) -> None:
    rule_dir = RULES_DIR / rule_id
    deps = DEPS.get(rule_id, {"pos": [], "neg": []})

    for variant in ("positive", "negative"):
        test_dir = rule_dir / "tests" / variant
        if not test_dir.exists():
            print(f"  SKIP {rule_id}/{variant} — directory missing")
            continue

        # Remove existing go.mod/go.sum to start clean
        for f in ("go.mod", "go.sum"):
            (test_dir / f).unlink(missing_ok=True)

        dep_list = deps.get("pos" if variant == "positive" else "neg", [])
        write_go_mod(test_dir, rule_id, variant, dep_list)

        print(f"  running go mod tidy in {test_dir.relative_to(RULES_DIR.parent.parent)}...")
        if not run_go_mod_tidy(test_dir):
            sys.exit(1)
        print(f"  ✓ go.sum generated")


def main() -> None:
    rules = sorted(r for r in DEPS)
    print(f"Processing {len(rules)} rules ({len(rules) * 2} test directories)...\n")

    for rule_id in rules:
        print(f"[{rule_id}]")
        process_rule(rule_id)
        print()

    print("All done ✓")


if __name__ == "__main__":
    main()
