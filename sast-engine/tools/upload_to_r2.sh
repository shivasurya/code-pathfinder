#!/bin/bash
set -e

# Upload Python stdlib registries to Cloudflare R2
# Usage: ./upload_to_r2.sh [temp_dir]

PYTHON_VERSIONS=("3.9" "3.10" "3.11" "3.12" "3.13" "3.14")
TEMP_DIR="${1:-/tmp/cpf-stdlib-registries}"
R2_BUCKET="code-pathfinder-assets"

# Check if required environment variables are set
if [ -z "$R2_ACCOUNT_ID" ] || [ -z "$R2_ACCESS_KEY_ID" ] || [ -z "$R2_SECRET_ACCESS_KEY" ]; then
    echo "Error: Missing R2 credentials"
    echo "Required environment variables:"
    echo "  - R2_ACCOUNT_ID"
    echo "  - R2_ACCESS_KEY_ID"
    echo "  - R2_SECRET_ACCESS_KEY"
    exit 1
fi

# Configure AWS CLI for R2
export AWS_ACCESS_KEY_ID="$R2_ACCESS_KEY_ID"
export AWS_SECRET_ACCESS_KEY="$R2_SECRET_ACCESS_KEY"
R2_ENDPOINT="https://${R2_ACCOUNT_ID}.r2.cloudflarestorage.com"

echo "========================================="
echo "Stdlib Registry Generation & Upload"
echo "========================================="
echo "Temp directory: $TEMP_DIR"
echo "R2 Bucket: $R2_BUCKET"
echo "R2 Endpoint: $R2_ENDPOINT"
echo ""

# Create temp directory
mkdir -p "$TEMP_DIR"

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Generate and upload for each Python version
for version in "${PYTHON_VERSIONS[@]}"; do
    echo "========================================="
    echo "Processing Python $version"
    echo "========================================="

    OUTPUT_DIR="$TEMP_DIR/registries/python${version}/stdlib/v1"

    # Check if Python version is available
    if ! command -v python${version} &> /dev/null; then
        echo "Warning: Python $version not found, skipping..."
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
        echo "Error: manifest.json not found for Python $version"
        exit 1
    fi

    # Count generated files
    FILE_COUNT=$(ls -1 "$OUTPUT_DIR"/*.json | wc -l)
    TOTAL_SIZE=$(du -sh "$OUTPUT_DIR" | cut -f1)
    echo "  ✓ Generated $FILE_COUNT files ($TOTAL_SIZE total)"

    echo ""
    echo "Step 3/3: Uploading to R2..."

    # Upload to R2 using AWS S3 sync
    aws s3 sync "$OUTPUT_DIR" \
        "s3://$R2_BUCKET/registries/python${version}/stdlib/v1/" \
        --endpoint-url "$R2_ENDPOINT" \
        --delete \
        --content-type "application/json" \
        --cache-control "public, max-age=3600"

    echo "  ✓ Uploaded Python $version registry"
    echo ""
done

echo "========================================="
echo "Upload Complete!"
echo "========================================="
echo "Registries are now available at:"
for version in "${PYTHON_VERSIONS[@]}"; do
    echo "  https://assets.codepathfinder.dev/registries/python${version}/stdlib/v1/manifest.json"
done
echo ""

# Clean up temp directory
if [ "$TEMP_DIR" = "/tmp/cpf-stdlib-registries" ]; then
    echo "Cleaning up temp directory..."
    rm -rf "$TEMP_DIR"
    echo "  ✓ Cleaned up $TEMP_DIR"
fi

echo ""
echo "Done! 🎉"
