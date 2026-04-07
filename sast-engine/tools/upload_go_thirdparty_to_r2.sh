#!/bin/bash
set -e

# Upload Go third-party registries to Cloudflare R2
#
# Usage: ./upload_go_thirdparty_to_r2.sh [temp_dir]
#
# This script:
#   1. Runs generate_go_thirdparty_registry.go to download modules and extract metadata
#   2. Uploads the generated JSON files to R2 under registries/go-thirdparty/v1/
#
# Required environment variables:
#   R2_ACCOUNT_ID, R2_ACCESS_KEY_ID, R2_SECRET_ACCESS_KEY

TEMP_DIR="${1:-/tmp/cpf-go-thirdparty-registries}"
R2_BUCKET="code-pathfinder-assets"
R2_PATH="registries/go-thirdparty/v1"

# ---------------------------------------------------------------------------
# Validate R2 credentials
# ---------------------------------------------------------------------------
if [ -z "$R2_ACCOUNT_ID" ] || [ -z "$R2_ACCESS_KEY_ID" ] || [ -z "$R2_SECRET_ACCESS_KEY" ]; then
    echo "Error: Missing R2 credentials"
    echo "Required environment variables:"
    echo "  - R2_ACCOUNT_ID"
    echo "  - R2_ACCESS_KEY_ID"
    echo "  - R2_SECRET_ACCESS_KEY"
    exit 1
fi

export AWS_ACCESS_KEY_ID="$R2_ACCESS_KEY_ID"
export AWS_SECRET_ACCESS_KEY="$R2_SECRET_ACCESS_KEY"
R2_ENDPOINT="https://${R2_ACCOUNT_ID}.r2.cloudflarestorage.com"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MODULE_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"   # sast-engine/
OUTPUT_DIR="$TEMP_DIR/out/go-thirdparty/v1"

echo "========================================="
echo "Go Third-Party Registry Generation & Upload"
echo "========================================="
echo "Temp directory : $TEMP_DIR"
echo "Output dir     : $OUTPUT_DIR"
echo "R2 Bucket      : $R2_BUCKET"
echo "R2 Path        : $R2_PATH"
echo "Go compiler    : $(go version)"
echo ""

mkdir -p "$OUTPUT_DIR"

# ---------------------------------------------------------------------------
# Step 1: Generate registry JSON
# ---------------------------------------------------------------------------
echo "Step 1/4: Generating Go third-party registry..."

(cd "$MODULE_DIR" && go run -tags cpf_generate_thirdparty_registry \
    tools/generate_go_thirdparty_registry.go \
    --packages-file tools/top1000.txt \
    --output-dir "$OUTPUT_DIR")

# Verify manifest was generated
if [ ! -f "$OUTPUT_DIR/manifest.json" ]; then
    echo "Error: manifest.json not generated"
    exit 1
fi

FILE_COUNT=$(find "$OUTPUT_DIR" -name "*.json" | wc -l)
TOTAL_SIZE=$(du -sh "$OUTPUT_DIR" | cut -f1)
echo ""
echo "  Generated $FILE_COUNT files ($TOTAL_SIZE total)"

# Show manifest stats
go run - <<'GOEOF' "$OUTPUT_DIR/manifest.json"
package main

import (
    "encoding/json"
    "fmt"
    "os"
)

func main() {
    data, _ := os.ReadFile(os.Args[1])
    var m struct {
        Packages []struct{} `json:"packages"`
    }
    json.Unmarshal(data, &m)
    fmt.Printf("  Packages: %d\n", len(m.Packages))
}
GOEOF

echo ""

# ---------------------------------------------------------------------------
# Step 2: Validate manifest JSON
# ---------------------------------------------------------------------------
echo "Step 2/4: Validating manifest..."

if ! python3 -c "import json; json.load(open('$OUTPUT_DIR/manifest.json'))" 2>/dev/null; then
    echo "Error: manifest.json is not valid JSON"
    exit 1
fi
echo "  manifest.json is valid JSON"
echo ""

# ---------------------------------------------------------------------------
# Step 3: Upload to R2
# ---------------------------------------------------------------------------
echo "Step 3/4: Uploading to R2..."

aws s3 sync "$OUTPUT_DIR" \
    "s3://$R2_BUCKET/$R2_PATH/" \
    --endpoint-url "$R2_ENDPOINT" \
    --delete \
    --content-type "application/json" \
    --cache-control "public, max-age=3600"

echo "  Uploaded to s3://$R2_BUCKET/$R2_PATH/"
echo ""

# ---------------------------------------------------------------------------
# Step 4: Verify public accessibility
# ---------------------------------------------------------------------------
echo "Step 4/4: Verifying public CDN accessibility..."

CDN_URL="https://assets.codepathfinder.dev/registries/go-thirdparty/v1/manifest.json"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$CDN_URL")
if [ "$STATUS" = "200" ]; then
    echo "  manifest.json is publicly accessible at:"
    echo "  $CDN_URL"
else
    echo "  Warning: manifest returned HTTP $STATUS (CDN propagation may be in progress)"
fi

echo ""
echo "========================================="
echo "Go Third-Party Registry Upload Complete!"
echo "========================================="
echo ""
echo "Registry available at:"
echo "  https://assets.codepathfinder.dev/registries/go-thirdparty/v1/manifest.json"
echo ""

# Clean up temp directory (only if using default path)
if [ "$TEMP_DIR" = "/tmp/cpf-go-thirdparty-registries" ]; then
    echo "Cleaning up temp directory..."
    rm -rf "$TEMP_DIR"
    echo "  Cleaned up $TEMP_DIR"
fi

echo "Done!"
