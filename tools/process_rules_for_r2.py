#!/usr/bin/env python3
"""
process_rules_for_r2.py - Process rule bundles for R2 distribution

Creates zip files per bundle with checksums and updates manifests.
"""

import argparse
import hashlib
import json
import os
import zipfile
from pathlib import Path
from typing import Dict, List, Tuple

class RuleProcessor:
    def __init__(self, rules_dir: str, output_dir: str, base_url: str, dry_run: bool = False):
        self.rules_dir = Path(rules_dir)
        self.output_dir = Path(output_dir)
        self.base_url = base_url.rstrip('/')
        self.dry_run = dry_run

    def process_all(self) -> None:
        """Main entry point: process all categories and bundles"""
        print(f"Processing rules from: {self.rules_dir}")
        print(f"Output directory: {self.output_dir}")
        print(f"Base URL: {self.base_url}")
        print(f"Dry run: {self.dry_run}\n")

        # Create output directory
        if not self.dry_run:
            self.output_dir.mkdir(parents=True, exist_ok=True)

        # Load global manifest
        global_manifest_path = self.rules_dir / "manifest.json"
        if not global_manifest_path.exists():
            raise FileNotFoundError(f"Global manifest not found: {global_manifest_path}")

        with open(global_manifest_path) as f:
            global_manifest = json.load(f)

        # Process each category
        categories = global_manifest.get("categories", [])
        processed_categories = []

        for category in categories:
            print(f"📦 Processing category: {category}")
            category_info = self.process_category(category)
            if category_info:
                processed_categories.append(category_info)
            print()

        # Update global manifest with computed data
        self.write_global_manifest(global_manifest, processed_categories)

        print("✅ Processing complete!")
        if self.dry_run:
            print("⚠️  DRY RUN: No files were created")

    def process_category(self, category: str) -> Dict:
        """Process a single category (e.g., docker, docker-compose)"""
        category_dir = self.rules_dir / category
        manifest_path = category_dir / "manifest.json"

        if not manifest_path.exists():
            print(f"  ⚠️  Skipping {category}: No manifest.json found")
            return None

        with open(manifest_path) as f:
            manifest = json.load(f)

        # Get bundle directories
        bundles = manifest.get("bundles", {})
        processed_bundles = {}

        for bundle_name, bundle_meta in bundles.items():
            print(f"  📁 Bundle: {bundle_name}")
            bundle_info = self.process_bundle(category, bundle_name, bundle_meta)
            if bundle_info:
                processed_bundles[bundle_name] = {**bundle_meta, **bundle_info}

        # Write category manifest with computed data
        output_category_dir = self.output_dir / category
        if not self.dry_run:
            output_category_dir.mkdir(parents=True, exist_ok=True)

            updated_manifest = {**manifest, "bundles": processed_bundles}
            with open(output_category_dir / "manifest.json", 'w') as f:
                json.dump(updated_manifest, f, indent=2)

        return {
            "category": category,
            "bundle_count": len(processed_bundles),
            "manifest_url": f"{self.base_url}/{category}/manifest.json"
        }

    def process_bundle(self, category: str, bundle_name: str, bundle_meta: Dict) -> Dict:
        """Process a single bundle: create zip and checksum"""
        bundle_dir = self.rules_dir / category / bundle_name

        if not bundle_dir.exists() or not bundle_dir.is_dir():
            print(f"    ⚠️  Bundle directory not found: {bundle_dir}")
            return None

        # Find all Python files
        py_files = list(bundle_dir.glob("*.py"))
        if not py_files:
            print(f"    ⚠️  No Python files in bundle")
            return None

        print(f"    📄 Files: {len(py_files)}")

        # Create zip file
        zip_filename = f"{bundle_name}.zip"
        zip_path = self.output_dir / category / zip_filename
        checksum_path = self.output_dir / category / f"{zip_filename}.sha256"

        if self.dry_run:
            # Just count files
            total_size = sum(f.stat().st_size for f in py_files)
            checksum = "DRY_RUN_CHECKSUM"
        else:
            # Actually create zip
            zip_path.parent.mkdir(parents=True, exist_ok=True)

            with zipfile.ZipFile(zip_path, 'w', zipfile.ZIP_DEFLATED) as zf:
                for py_file in sorted(py_files):
                    arcname = py_file.name  # Store only filename, not full path
                    zf.write(py_file, arcname=arcname)

            total_size = zip_path.stat().st_size

            # Calculate checksum
            checksum = self.calculate_sha256(zip_path)

            # Write checksum file
            with open(checksum_path, 'w') as f:
                f.write(f"{checksum}  {zip_filename}\n")

        print(f"    📦 Zip: {zip_filename} ({self.format_size(total_size)})")
        print(f"    🔒 SHA256: {checksum[:16]}...")

        return {
            "file_count": len(py_files),
            "zip_size": total_size,
            "checksum": checksum,
            "download_url": f"{self.base_url}/{category}/{zip_filename}"
        }

    def calculate_sha256(self, file_path: Path) -> str:
        """Calculate SHA256 checksum of a file"""
        sha256 = hashlib.sha256()
        with open(file_path, 'rb') as f:
            for chunk in iter(lambda: f.read(8192), b''):
                sha256.update(chunk)
        return sha256.hexdigest()

    def write_global_manifest(self, original: Dict, categories: List[Dict]) -> None:
        """Write updated global manifest with computed metadata"""
        updated = {
            **original,
            "categories_info": categories,
            "base_url": self.base_url
        }

        output_path = self.output_dir / "manifest.json"
        if not self.dry_run:
            with open(output_path, 'w') as f:
                json.dump(updated, f, indent=2)
            print(f"📝 Global manifest: {output_path}")

    @staticmethod
    def format_size(size: int) -> str:
        """Format bytes as human-readable"""
        for unit in ['B', 'KB', 'MB', 'GB']:
            if size < 1024:
                return f"{size:.1f} {unit}"
            size /= 1024
        return f"{size:.1f} TB"

def main():
    parser = argparse.ArgumentParser(
        description="Process rule bundles for R2 distribution"
    )
    parser.add_argument(
        '--rules-dir',
        default='./rules',
        help='Rules directory (default: ./rules)'
    )
    parser.add_argument(
        '--output-dir',
        default='./dist/rules',
        help='Output directory (default: ./dist/rules)'
    )
    parser.add_argument(
        '--base-url',
        default='https://assets.codepathfinder.dev/rules',
        help='Base URL for downloads'
    )
    parser.add_argument(
        '--dry-run',
        action='store_true',
        help='Preview without creating files'
    )

    args = parser.parse_args()

    processor = RuleProcessor(
        rules_dir=args.rules_dir,
        output_dir=args.output_dir,
        base_url=args.base_url,
        dry_run=args.dry_run
    )

    try:
        processor.process_all()
    except Exception as e:
        print(f"\n❌ Error: {e}")
        import traceback
        traceback.print_exc()
        exit(1)

if __name__ == '__main__':
    main()
