#!/bin/bash
set -e

# Upload third-party type registries to Cloudflare R2
# Usage: ./upload_thirdparty_to_r2.sh [temp_dir]
#
# This script:
#   1. Clones typeshed and django-stubs repos
#   2. Installs PEP 561 packages (flask, sqlalchemy, etc.)
#   3. Runs the typeshed-converter to generate JSON registries
#   4. Uploads to R2 under thirdparty/v1/

TEMP_DIR="${1:-/tmp/cpf-thirdparty-registries}"
R2_BUCKET="code-pathfinder-assets"
PYTHON="${PYTHON:-python3}"

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

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONVERTER_DIR="$SCRIPT_DIR/typeshed-converter"
OUTPUT_DIR="$TEMP_DIR/output/thirdparty/v1"
SOURCES_DIR="$TEMP_DIR/sources"

echo "========================================="
echo "Third-Party Registry Generation & Upload"
echo "========================================="
echo "Temp directory: $TEMP_DIR"
echo "R2 Bucket:      $R2_BUCKET"
echo "Python:         $($PYTHON --version)"
echo ""

# Create directories
mkdir -p "$OUTPUT_DIR" "$SOURCES_DIR"

# ── Step 1: Clone source repositories ─────────────────────────────────────

echo "Step 1/5: Cloning source repositories..."

# Clone typeshed (shallow, only stubs/)
if [ ! -d "$SOURCES_DIR/typeshed" ]; then
    echo "  Cloning typeshed..."
    git clone --depth 1 https://github.com/python/typeshed.git "$SOURCES_DIR/typeshed"
fi
echo "  ✓ typeshed ready ($(ls -d "$SOURCES_DIR/typeshed/stubs"/*/ | wc -l) packages)"

# Clone django-stubs
if [ ! -d "$SOURCES_DIR/django-stubs" ]; then
    echo "  Cloning django-stubs..."
    git clone --depth 1 https://github.com/typeddjango/django-stubs.git "$SOURCES_DIR/django-stubs"
fi
echo "  ✓ django-stubs ready"

# Clone djangorestframework-stubs
if [ ! -d "$SOURCES_DIR/djangorestframework-stubs" ]; then
    echo "  Cloning djangorestframework-stubs..."
    git clone --depth 1 https://github.com/typeddjango/djangorestframework-stubs.git "$SOURCES_DIR/djangorestframework-stubs"
fi
echo "  ✓ djangorestframework-stubs ready"
echo ""

# ── Step 2: Install PEP 561 packages ──────────────────────────────────────

echo "Step 2/5: Installing PEP 561 packages..."

VENV_DIR="$SOURCES_DIR/venv"
if [ ! -d "$VENV_DIR" ]; then
    $PYTHON -m venv "$VENV_DIR"
fi
# shellcheck disable=SC1091
source "$VENV_DIR/bin/activate"

# Install PEP 561 packages with inline type annotations
pip install --quiet \
    flask sqlalchemy fastapi starlette pydantic \
    werkzeug jinja2 click httpx redis celery

SITE_PACKAGES=$(python -c "import sysconfig; print(sysconfig.get_path('purelib'))")
echo "  ✓ PEP 561 packages installed to $SITE_PACKAGES"
echo ""

# ── Step 3: Install converter dependencies ────────────────────────────────

echo "Step 3/5: Installing converter dependencies..."
pip install --quiet -r "$CONVERTER_DIR/requirements.txt"
echo "  ✓ Converter dependencies installed"
echo ""

# ── Step 4: Generate registry JSON ────────────────────────────────────────

echo "Step 4/5: Generating third-party registry..."

# Build a CI-specific sources.yaml pointing to cloned repos
CI_SOURCES="$TEMP_DIR/ci-sources.yaml"
cat > "$CI_SOURCES" <<YAML
sources:
  # Typeshed stubs (195+ packages)
  - type: typeshed
    path: $SOURCES_DIR/typeshed/stubs

  # Django stubs
  - type: pyi_repo
    path: $SOURCES_DIR/django-stubs/django-stubs
    package: django
    source_name: django-stubs

  # Django REST Framework stubs
  - type: pyi_repo
    path: $SOURCES_DIR/djangorestframework-stubs/rest_framework-stubs
    package: rest_framework
    source_name: djangorestframework-stubs

  # PEP 561 packages
  - type: pep561
    package: flask
    path: $SITE_PACKAGES/flask

  - type: pep561
    package: sqlalchemy
    path: $SITE_PACKAGES/sqlalchemy

  - type: pep561
    package: fastapi
    path: $SITE_PACKAGES/fastapi

  - type: pep561
    package: starlette
    path: $SITE_PACKAGES/starlette

  - type: pep561
    package: pydantic
    path: $SITE_PACKAGES/pydantic

  - type: pep561
    package: werkzeug
    path: $SITE_PACKAGES/werkzeug

  - type: pep561
    package: jinja2
    path: $SITE_PACKAGES/jinja2

  - type: pep561
    package: click
    path: $SITE_PACKAGES/click

  - type: pep561
    package: httpx
    path: $SITE_PACKAGES/httpx

  - type: pep561
    package: redis
    path: $SITE_PACKAGES/redis

  - type: pep561
    package: celery
    path: $SITE_PACKAGES/celery
YAML

python "$CONVERTER_DIR/convert.py" \
    --config "$CI_SOURCES" \
    --output "$OUTPUT_DIR" \
    --base-url "https://assets.codepathfinder.dev/registries/thirdparty/v1" \
    --verbose

# Verify manifest was generated
if [ ! -f "$OUTPUT_DIR/manifest.json" ]; then
    echo "Error: manifest.json not generated"
    exit 1
fi

FILE_COUNT=$(find "$OUTPUT_DIR" -name "*.json" | wc -l)
TOTAL_SIZE=$(du -sh "$OUTPUT_DIR" | cut -f1)
echo ""
echo "  ✓ Generated $FILE_COUNT files ($TOTAL_SIZE total)"

# Show manifest stats
python -c "
import json, sys
m = json.load(open('$OUTPUT_DIR/manifest.json'))
s = m.get('statistics', {})
print(f'  Modules:    {s.get(\"total_modules\", 0)}')
print(f'  Functions:  {s.get(\"total_functions\", 0)}')
print(f'  Classes:    {s.get(\"total_classes\", 0)}')
print(f'  Constants:  {s.get(\"total_constants\", 0)}')
print(f'  Attributes: {s.get(\"total_attributes\", 0)}')
"
echo ""

# ── Step 5: Upload to R2 ─────────────────────────────────────────────────

echo "Step 5/5: Uploading to R2..."

aws s3 sync "$OUTPUT_DIR" \
    "s3://$R2_BUCKET/registries/thirdparty/v1/" \
    --endpoint-url "$R2_ENDPOINT" \
    --delete \
    --content-type "application/json" \
    --cache-control "public, max-age=3600"

echo "  ✓ Uploaded to R2"
echo ""

# Deactivate venv
deactivate

echo "========================================="
echo "Third-Party Registry Upload Complete!"
echo "========================================="
echo "Registry available at:"
echo "  https://assets.codepathfinder.dev/registries/thirdparty/v1/manifest.json"
echo ""

# Clean up temp directory
if [ "$TEMP_DIR" = "/tmp/cpf-thirdparty-registries" ]; then
    echo "Cleaning up temp directory..."
    rm -rf "$TEMP_DIR"
    echo "  ✓ Cleaned up $TEMP_DIR"
fi

echo ""
echo "Done!"
