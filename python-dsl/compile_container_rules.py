#!/usr/bin/env python3
"""
Compile container security rules to JSON IR for Go executor.

This script scans the rules/docker/ and rules/docker-compose/ directories,
imports all rule files, and compiles them into a single JSON file that can
be loaded by the Go executor.
"""

import importlib.util
import json
import sys
from pathlib import Path

# Add python-dsl to path for imports
sys.path.insert(0, str(Path(__file__).parent))

from rules import container_ir
from rules import container_decorators


def discover_and_import_rules(rules_dir: Path, rule_type: str):
    """
    Discover and import all rule files from a directory.

    This function imports the rule files, which causes their decorators to
    register the rules in the global registries.

    Args:
        rules_dir: Directory containing rule Python files
        rule_type: 'dockerfile' or 'compose' (for logging)

    Returns:
        Number of files imported
    """
    imported_count = 0

    if not rules_dir.exists():
        print(f"Warning: {rules_dir} does not exist")
        return imported_count

    # Find all Python files except __init__.py
    rule_files = [f for f in rules_dir.glob("*.py") if f.name != "__init__.py"]

    print(f"\n{rule_type.upper()} rule files in {rules_dir.name}/:")

    for rule_file in sorted(rule_files):
        try:
            # Import the module dynamically
            module_name = f"rules_{rule_type}_{rule_file.stem}"
            spec = importlib.util.spec_from_file_location(module_name, rule_file)
            if spec and spec.loader:
                module = importlib.util.module_from_spec(spec)
                spec.loader.exec_module(module)
                imported_count += 1
                print(f"  ✓ {rule_file.name}")

        except Exception as e:
            print(f"  ✗ Failed to import {rule_file.name}: {e}")
            import traceback
            traceback.print_exc()

    return imported_count


def main():
    """Compile all container rules to JSON IR."""
    print("=" * 70)
    print(" " * 15 + "Compiling Container Security Rules")
    print("=" * 70)

    # Determine project root (go up from python-dsl/)
    project_root = Path(__file__).parent.parent
    rules_root = project_root / "rules"

    # Import Dockerfile rules (decorator registers them automatically)
    dockerfile_dir = rules_root / "docker"
    dockerfile_count = discover_and_import_rules(dockerfile_dir, "dockerfile")

    # Import Compose rules (decorator registers them automatically)
    compose_dir = rules_root / "docker-compose"
    compose_count = discover_and_import_rules(compose_dir, "compose")

    # Get rules from global registries
    dockerfile_rules = container_decorators._dockerfile_rules
    compose_rules = container_decorators._compose_rules

    total_rules = len(dockerfile_rules) + len(compose_rules)

    print(f"\n{'='*70}")
    print(f"Files imported: {dockerfile_count + compose_count}")
    print(f"Rules registered: {total_rules}")
    print(f"  - Dockerfile rules: {len(dockerfile_rules)}")
    print(f"  - Compose rules: {len(compose_rules)}")
    print(f"{'='*70}\n")

    if total_rules == 0:
        print("❌ No rules registered! Check that:")
        print(f"   1. Rule files exist in:")
        print(f"      - {dockerfile_dir}")
        print(f"      - {compose_dir}")
        print(f"   2. Rule files use @dockerfile_rule or @compose_rule decorators")
        print(f"   3. No import errors occurred above")
        return 1

    # List all registered rules
    print("Registered Dockerfile rules:")
    for rule_def in dockerfile_rules:
        print(f"  - {rule_def.metadata.id}: {rule_def.metadata.name}")

    print("\nRegistered Compose rules:")
    for rule_def in compose_rules:
        print(f"  - {rule_def.metadata.id}: {rule_def.metadata.name}")

    # Compile to JSON IR
    print(f"\n{'='*70}")
    print("Compiling to JSON IR...")
    json_ir = container_ir.compile_all_rules()

    # Write to file in python-dsl directory
    output_path = Path(__file__).parent / "compiled_rules.json"
    with open(output_path, "w") as f:
        json.dump(json_ir, f, indent=2)

    print(f"\n✅ Rules compiled successfully!")
    print(f"{'='*70}")
    print(f"Output: {output_path}")
    print(f"  - Dockerfile rules: {len(json_ir.get('dockerfile', []))}")
    print(f"  - Compose rules: {len(json_ir.get('compose', []))}")

    # Also copy to sast-engine for testing
    sast_output = project_root / "sast-engine" / "compiled_rules.json"
    with open(sast_output, "w") as f:
        json.dump(json_ir, f, indent=2)
    print(f"  - Copied to: {sast_output}")
    print(f"{'='*70}")

    return 0


if __name__ == "__main__":
    sys.exit(main())
