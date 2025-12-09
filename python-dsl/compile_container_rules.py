#!/usr/bin/env python3
"""
Compile container security rules to JSON IR for Go executor.

This script compiles all Dockerfile and docker-compose rules into a single
JSON file that can be loaded by the Go executor.
"""

import json
import sys
from pathlib import Path

# Add rules directory to path
sys.path.insert(0, str(Path(__file__).parent))

from rules import container_ir
from rules.dockerfile_rules import DOCKERFILE_RULES
from rules.compose_rules import COMPOSE_RULES


def main():
    """Compile all container rules to JSON IR."""
    print("Compiling container security rules...")

    print(f"\nDockerfile rules: {len(DOCKERFILE_RULES)}")
    for rule_func in DOCKERFILE_RULES:
        print(f"  - {rule_func.__name__}")

    print(f"\nCompose rules: {len(COMPOSE_RULES)}")
    for rule_func in COMPOSE_RULES:
        print(f"  - {rule_func.__name__}")

    print(f"\nTotal rules: {len(DOCKERFILE_RULES) + len(COMPOSE_RULES)}")

    # Compile to JSON IR (decorators already registered the rules)
    json_ir = container_ir.compile_all_rules()

    # Write to file
    output_path = Path(__file__).parent / "compiled_rules.json"
    with open(output_path, "w") as f:
        json.dump(json_ir, f, indent=2)

    print(f"\n✅ Rules compiled successfully to: {output_path}")
    print(f"   Dockerfile rules: {len(json_ir.get('dockerfile', []))}")
    print(f"   Compose rules: {len(json_ir.get('compose', []))}")

    return 0


if __name__ == "__main__":
    sys.exit(main())
