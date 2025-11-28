#!/bin/bash
set -e

# Test script to validate stdlib registry generation locally
# This does NOT upload to R2, just validates generation works

PYTHON_VERSIONS=("3.14")  # Test with one version for speed
TEST_DIR="/tmp/cpf-stdlib-test-$(date +%s)"

echo "========================================="
echo "Testing Stdlib Registry Generation"
echo "========================================="
echo "Test directory: $TEST_DIR"
echo ""

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Test each Python version
for version in "${PYTHON_VERSIONS[@]}"; do
    echo "========================================="
    echo "Testing Python $version"
    echo "========================================="

    OUTPUT_DIR="$TEST_DIR/registries/python${version}/stdlib/v1"

    # Check if Python version is available
    if ! command -v python${version} &> /dev/null; then
        echo "❌ Python $version not found, skipping..."
        echo ""
        continue
    fi

    echo "Step 1/3: Generating stdlib registry..."
    python${version} "$SCRIPT_DIR/generate_stdlib_registry.py" --all \
        --output-dir "$OUTPUT_DIR" \
        --verbose

    echo ""
    echo "Step 2/3: Verifying generated files..."

    # Check if manifest exists
    if [ ! -f "$OUTPUT_DIR/manifest.json" ]; then
        echo "❌ manifest.json not found"
        exit 1
    fi
    echo "  ✓ manifest.json exists"

    # Count generated files
    FILE_COUNT=$(ls -1 "$OUTPUT_DIR"/*.json 2>/dev/null | wc -l)
    echo "  ✓ Generated $FILE_COUNT JSON files"

    # Check total size
    TOTAL_SIZE=$(du -sh "$OUTPUT_DIR" | cut -f1)
    echo "  ✓ Total size: $TOTAL_SIZE"

    # Validate manifest JSON
    if ! cat "$OUTPUT_DIR/manifest.json" | jq empty 2>/dev/null; then
        echo "❌ manifest.json is not valid JSON"
        exit 1
    fi
    echo "  ✓ manifest.json is valid JSON"

    # Check manifest contains expected fields
    BASE_URL=$(cat "$OUTPUT_DIR/manifest.json" | jq -r '.base_url')
    if [[ "$BASE_URL" != *"assets.codepathfinder.dev"* ]]; then
        echo "❌ manifest.json has incorrect base_url: $BASE_URL"
        exit 1
    fi
    echo "  ✓ base_url is correct: $BASE_URL"

    # Check module count
    MODULE_COUNT=$(cat "$OUTPUT_DIR/manifest.json" | jq '.modules | length')
    echo "  ✓ manifest lists $MODULE_COUNT modules"

    # Validate a sample module file
    SAMPLE_MODULE=$(ls "$OUTPUT_DIR"/*_stdlib.json 2>/dev/null | head -1)
    if [ -n "$SAMPLE_MODULE" ]; then
        if ! cat "$SAMPLE_MODULE" | jq empty 2>/dev/null; then
            echo "❌ Sample module file is not valid JSON: $SAMPLE_MODULE"
            exit 1
        fi
        MODULE_NAME=$(basename "$SAMPLE_MODULE" _stdlib.json)
        echo "  ✓ Sample module '$MODULE_NAME' is valid JSON"
    fi

    echo ""
    echo "Step 3/3: Testing manifest structure..."

    # Extract and verify manifest fields
    PYTHON_VER=$(cat "$OUTPUT_DIR/manifest.json" | jq -r '.python_version.full')
    SCHEMA_VER=$(cat "$OUTPUT_DIR/manifest.json" | jq -r '.schema_version')
    TOTAL_FUNCS=$(cat "$OUTPUT_DIR/manifest.json" | jq -r '.statistics.total_functions')
    TOTAL_CLASSES=$(cat "$OUTPUT_DIR/manifest.json" | jq -r '.statistics.total_classes')

    echo "  ✓ Python version: $PYTHON_VER"
    echo "  ✓ Schema version: $SCHEMA_VER"
    echo "  ✓ Total functions: $TOTAL_FUNCS"
    echo "  ✓ Total classes: $TOTAL_CLASSES"

    echo ""
    echo "✅ Python $version registry generation test PASSED"
    echo ""
done

echo "========================================="
echo "Test Complete!"
echo "========================================="
echo ""
echo "Generated files are in: $TEST_DIR"
echo ""
echo "To inspect the output:"
echo "  ls -lh $TEST_DIR/registries/python3.14/stdlib/v1/"
echo "  cat $TEST_DIR/registries/python3.14/stdlib/v1/manifest.json | jq ."
echo ""
echo "To clean up:"
echo "  rm -rf $TEST_DIR"
echo ""
echo "✅ All tests passed! Ready for R2 upload."
