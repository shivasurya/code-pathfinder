#!/usr/bin/env bash
# Smoke test for the version-update-check chain.
# Usage: smoke-update-check.sh <expected-tag>
# Run manually after PR-01..PR-05 have all merged and the binary is on main.
set -euo pipefail

CDN_URL="https://assets.codepathfinder.dev/pathfinder/latest.json"
EXPECTED_TAG="${1:?usage: smoke-update-check.sh <expected-tag>}"

echo "1. Checking CDN serves the expected tag..."
ACTUAL=$(curl -fsSL "$CDN_URL" | jq -r '.latest.version')
if [[ "$ACTUAL" != "$EXPECTED_TAG" ]]; then
  echo "FAIL: CDN says $ACTUAL, expected $EXPECTED_TAG"
  exit 1
fi
echo "  OK ($ACTUAL)"

echo "2. Checking a stale binary sees an upgrade notice..."
go build -ldflags="-X github.com/shivasurya/code-pathfinder/sast-engine/cmd.Version=0.0.1" \
  -o /tmp/pathfinder-stale ./sast-engine
output=$(/tmp/pathfinder-stale version 2>&1)
if ! grep -q "Update available" <<< "$output"; then
  echo "FAIL: stale binary did not render upgrade notice"
  echo "$output"
  exit 1
fi
echo "  OK"

echo "3. Checking a current binary sees no notice..."
go build -ldflags="-X github.com/shivasurya/code-pathfinder/sast-engine/cmd.Version=$EXPECTED_TAG" \
  -o /tmp/pathfinder-current ./sast-engine
output=$(/tmp/pathfinder-current version 2>&1)
if grep -q "Update available" <<< "$output"; then
  echo "FAIL: current binary rendered upgrade notice"
  echo "$output"
  exit 1
fi
echo "  OK"

echo "All automated smoke checks passed."
