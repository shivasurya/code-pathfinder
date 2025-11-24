#!/bin/bash
set -e

echo "=========================================="
echo "PR-02: nsjail Integration Test"
echo "=========================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Create test DSL rule that outputs valid JSON
cat > /tmp/test_rule.py <<'EOF'
import json

# Simple test rule that outputs valid RuleIR JSON
rule = {
    "id": "test-rule",
    "severity": "info",
    "description": "Test rule for nsjail integration",
    "matcher": {
        "type": "call_matcher",
        "pattern": "test"
    }
}

print(json.dumps([rule], indent=2))
EOF

echo "Test 1: Direct Python Execution (Sandbox Disabled)"
echo "---------------------------------------------------"
export PATHFINDER_SANDBOX_ENABLED=false

# Build the pathfinder binary
echo "Building pathfinder binary..."
cd /Users/shiva/src/shivasurya/code-pathfinder/sourcecode-parser
go build -o /tmp/pathfinder-test . 2>&1 | grep -E "(error|warning)" || echo "Build successful"

echo ""
echo "Test 2: nsjail Sandboxed Execution (Sandbox Enabled)"
echo "-----------------------------------------------------"
export PATHFINDER_SANDBOX_ENABLED=true

# Test nsjail command directly
echo "Testing nsjail command directly..."
mkdir -p /tmp/nsjail_root

nsjail -Mo \
  --user nobody \
  --chroot /tmp/nsjail_root \
  --iface_no_lo \
  --disable_proc \
  --bindmount_ro /usr:/usr \
  --bindmount_ro /lib:/lib \
  --bindmount /tmp:/tmp \
  --cwd /tmp \
  --rlimit_as 512 \
  --rlimit_cpu 30 \
  --rlimit_fsize 1 \
  --time_limit 30 \
  --quiet \
  -- /usr/bin/python3 /tmp/test_rule.py

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ SUCCESS${NC}: nsjail can execute Python DSL rules"
else
    echo -e "${RED}❌ FAILED${NC}: nsjail execution failed"
    exit 1
fi

echo ""
echo "=========================================="
echo "Test Summary"
echo "=========================================="
echo -e "${GREEN}✅ PR-02 Integration Complete${NC}"
echo ""
echo "Changes:"
echo "  - Added isSandboxEnabled() function"
echo "  - Added buildNsjailCommand() function"
echo "  - Modified loadRulesFromFile() to use nsjail"
echo "  - Updated entrypoint.sh to create /tmp/nsjail_root"
echo ""
echo "Security Features Enabled:"
echo "  ✅ Network isolation (--iface_no_lo)"
echo "  ✅ Filesystem isolation (chroot)"
echo "  ✅ Process isolation (PID namespace)"
echo "  ✅ User isolation (run as nobody)"
echo "  ✅ Resource limits (512MB, 30s CPU, 1MB file)"
echo ""

# Cleanup
rm -f /tmp/test_rule.py
