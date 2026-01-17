#!/bin/bash
# upload_rules_to_r2.sh - Upload processed rules to R2

set -euo pipefail

DIST_DIR="${1:-./dist/rules}"
BUCKET="code-pathfinder-assets"
R2_PREFIX="rules"

# Convert to absolute path for consistent path operations
DIST_DIR=$(cd "$DIST_DIR" 2>/dev/null && pwd || echo "$DIST_DIR")

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check prerequisites
check_prerequisites() {
    echo "🔍 Checking prerequisites..."

    # Check AWS CLI
    if ! command -v aws &> /dev/null; then
        echo -e "${RED}❌ aws CLI not found${NC}"
        echo "Install: https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html"
        exit 1
    fi

    # Check environment variables
    if [ -z "$AWS_ACCESS_KEY_ID" ] || [ -z "$AWS_SECRET_ACCESS_KEY" ]; then
        echo -e "${RED}❌ R2 credentials not set${NC}"
        echo "Required: AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY"
        exit 1
    fi

    # Check R2 account ID
    if [ -z "$R2_ACCOUNT_ID" ]; then
        echo -e "${RED}❌ R2_ACCOUNT_ID not set${NC}"
        echo "Required: R2_ACCOUNT_ID (Cloudflare R2 Account ID)"
        echo ""
        echo "This script uses the same R2 credentials as stdlib uploads."
        echo "The R2_ACCOUNT_ID secret should already be configured in GitHub Actions."
        exit 1
    fi

    # Validate R2 account ID format (alphanumeric)
    if [[ ! "$R2_ACCOUNT_ID" =~ ^[a-z0-9]+$ ]]; then
        echo -e "${RED}❌ Invalid R2_ACCOUNT_ID format${NC}"
        echo "Expected: alphanumeric string (e.g., abc123def456)"
        echo "Got: $R2_ACCOUNT_ID"
        exit 1
    fi

    # Construct R2 endpoint from account ID (same as stdlib workflow)
    R2_ENDPOINT="https://${R2_ACCOUNT_ID}.r2.cloudflarestorage.com"

    # Check dist directory
    if [ ! -d "$DIST_DIR" ]; then
        echo -e "${RED}❌ Distribution directory not found: $DIST_DIR${NC}"
        echo "Run process_rules_for_r2.py first"
        exit 1
    fi

    echo -e "${GREEN}✅ Prerequisites OK${NC}\n"
}

# Upload a single file to R2
upload_file() {
    local file_path="$1"
    local s3_key="$2"
    local content_type="$3"
    local cache_control="$4"

    local filename=$(basename "$file_path")
    echo "  📤 Uploading: $filename"

    aws s3 cp "$file_path" "s3://$BUCKET/$s3_key" \
        --endpoint-url "$R2_ENDPOINT" \
        --content-type "$content_type" \
        --cache-control "$cache_control" \
        --acl public-read

    echo -e "  ${GREEN}✅ Uploaded${NC}\n"
}

# Upload manifest files (short cache)
upload_manifests() {
    echo "📝 Uploading manifest files..."

    # Global manifest
    if [ -f "$DIST_DIR/manifest.json" ]; then
        upload_file \
            "$DIST_DIR/manifest.json" \
            "$R2_PREFIX/manifest.json" \
            "application/json" \
            "max-age=3600, public"  # 1 hour cache
    fi

    # Category manifests (all manifests in subdirectories)
    find "$DIST_DIR" -mindepth 2 -type f -name "manifest.json" | while IFS= read -r manifest; do
        # Calculate relative path using parameter expansion (more portable)
        relative_path="${manifest#$DIST_DIR/}"
        upload_file \
            "$manifest" \
            "$R2_PREFIX/$relative_path" \
            "application/json" \
            "max-age=3600, public"
    done
}

# Upload zip bundles (long cache, immutable)
upload_bundles() {
    echo "📦 Uploading zip bundles..."

    find "$DIST_DIR" -type f -name "*.zip" | while IFS= read -r zipfile; do
        # Calculate relative path using parameter expansion (more portable)
        relative_path="${zipfile#$DIST_DIR/}"
        upload_file \
            "$zipfile" \
            "$R2_PREFIX/$relative_path" \
            "application/zip" \
            "max-age=86400, immutable, public"  # 24 hour cache

        # Upload checksum file
        checksum_file="${zipfile}.sha256"
        if [ -f "$checksum_file" ]; then
            checksum_relative="${checksum_file#$DIST_DIR/}"
            upload_file \
                "$checksum_file" \
                "$R2_PREFIX/$checksum_relative" \
                "text/plain" \
                "max-age=86400, immutable, public"
        fi
    done
}

# Verify uploads
verify_uploads() {
    echo "🔍 Verifying uploads..."

    # Check if global manifest is accessible
    MANIFEST_URL="https://assets.codepathfinder.dev/rules/manifest.json"

    if curl -f -s "$MANIFEST_URL" > /dev/null; then
        echo -e "${GREEN}✅ Global manifest accessible${NC}"
    else
        echo -e "${RED}❌ Global manifest not accessible: $MANIFEST_URL${NC}"
        exit 1
    fi

    echo -e "${GREEN}✅ Verification complete${NC}\n"
}

# Main execution
main() {
    echo "================================================"
    echo "  R2 Rules Upload Script"
    echo "================================================"
    echo ""

    check_prerequisites

    echo "📂 Distribution directory: $DIST_DIR"
    echo "🪣  R2 Bucket: $BUCKET"
    echo ""

    # Count files (trim whitespace from wc output)
    manifest_count=$(find "$DIST_DIR" -name "manifest.json" | wc -l | tr -d ' ')
    zip_count=$(find "$DIST_DIR" -name "*.zip" | wc -l | tr -d ' ')
    echo "Files to upload:"
    echo "  - Manifests: $manifest_count"
    echo "  - Zip bundles: $zip_count"
    echo ""

    # Skip confirmation if UPLOAD_CONFIRMED is set (for CI)
    if [ "$UPLOAD_CONFIRMED" != "yes" ]; then
        read -p "Proceed with upload? (y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            echo "Upload cancelled"
            exit 0
        fi
    fi

    upload_manifests
    upload_bundles
    verify_uploads

    echo "================================================"
    echo -e "${GREEN}✅ All uploads complete!${NC}"
    echo "================================================"
}

main
