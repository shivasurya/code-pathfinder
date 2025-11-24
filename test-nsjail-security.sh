#!/bin/bash
set -e

echo "=========================================="
echo "nsjail Security Validation Test Suite"
echo "=========================================="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

PASS_COUNT=0
FAIL_COUNT=0
TOTAL_TESTS=10

# Helper functions
pass() {
    echo -e "${GREEN}✅ PASS${NC}: $1"
    ((PASS_COUNT++))
}

fail() {
    echo -e "${RED}❌ FAIL${NC}: $1"
    ((FAIL_COUNT++))
}

info() {
    echo -e "${YELLOW}ℹ️  INFO${NC}: $1"
}

# Check if running with CAP_SYS_ADMIN
echo "Checking prerequisites..."
if ! podman run --rm --cap-add=SYS_ADMIN --entrypoint sh pathfinder:sandbox-test -c "echo 'Container access OK'" &>/dev/null; then
    echo "ERROR: Cannot access pathfinder:sandbox-test image"
    echo "Run: podman build -t pathfinder:sandbox-test ."
    exit 1
fi
echo ""

# Helper function to run sandboxed test
run_test() {
    local test_code="$1"
    podman run --rm --cap-add=SYS_ADMIN --entrypoint sh pathfinder:sandbox-test -c "
cat > /tmp/test.py <<'PYEOF'
$test_code
PYEOF

mkdir -p /tmp/nsjail_root
nsjail -Mo --user nobody --chroot /tmp/nsjail_root --iface_no_lo --disable_proc \
  --bindmount_ro /usr:/usr --bindmount_ro /lib:/lib --bindmount /tmp:/tmp --cwd /tmp \
  --rlimit_as 512 --rlimit_cpu 30 --rlimit_fsize 1 --time_limit 30 --quiet \
  -- /usr/bin/python3 /tmp/test.py 2>&1 | tail -5
" 2>&1
}

# Test 1: Network Access Blocking
echo "Test 1: Network Access Blocking"
echo "--------------------------------"
result=$(run_test "
import socket, sys
try:
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.settimeout(2)
    s.connect(('8.8.8.8', 53))
    print('FAIL')
    sys.exit(1)
except OSError as e:
    print('PASS')
    sys.exit(0)
")

if [[ "$result" =~ "PASS" ]]; then
    pass "Network access completely blocked"
else
    fail "Network access not blocked"
fi
echo ""

# Test 2: Sensitive File Access Blocking
echo "Test 2: Sensitive File Access Blocking"
echo "---------------------------------------"
result=$(run_test "
import sys
sensitive_files = ['/etc/passwd', '/etc/shadow', '/root/.bashrc', '/proc/self/environ', '/root/.ssh/id_rsa']
failed = []
for filepath in sensitive_files:
    try:
        with open(filepath, 'r') as f:
            f.read()
        failed.append(filepath)
    except (FileNotFoundError, PermissionError):
        pass

if failed:
    print(f'FAIL:{','.join(failed)}')
    sys.exit(1)
else:
    print('PASS')
    sys.exit(0)
")

if [[ "$result" =~ "PASS" ]]; then
    pass "All sensitive files blocked (/etc/passwd, /etc/shadow, /root/.ssh, /proc/self/environ)"
else
    fail "Some files accessible: $result"
fi
echo ""

# Test 3: PID Namespace Isolation
echo "Test 3: PID Namespace Isolation"
echo "--------------------------------"
result=$(run_test "
import os, sys
my_pid = os.getpid()
if my_pid == 1:
    print('PASS')
    sys.exit(0)
else:
    print(f'FAIL:PID={my_pid}')
    sys.exit(1)
")

if [[ "$result" =~ "PASS" ]]; then
    pass "PID namespace isolated (process sees itself as PID 1)"
else
    fail "PID namespace not isolated: $result"
fi
echo ""

# Test 4: Filesystem Write Restrictions
echo "Test 4: Filesystem Write Restrictions"
echo "--------------------------------------"
result=$(run_test "
import sys
restricted_locations = ['/etc/test', '/usr/test', '/test']
failed = []
for loc in restricted_locations:
    try:
        with open(loc, 'w') as f:
            f.write('test')
        failed.append(loc)
    except (OSError, PermissionError, FileNotFoundError):
        pass

if failed:
    print(f'FAIL:{','.join(failed)}')
    sys.exit(1)
else:
    print('PASS')
    sys.exit(0)
")

if [[ "$result" =~ "PASS" ]]; then
    pass "Filesystem read-only (cannot write to /, /etc, /usr)"
else
    fail "Can write to restricted locations: $result"
fi
echo ""

# Test 5: Environment Variable Exposure
echo "Test 5: Environment Variable Exposure"
echo "--------------------------------------"
result=$(run_test "
import os, sys
env_vars = list(os.environ.keys())
allowed = ['LC_CTYPE']
unexpected = [v for v in env_vars if v not in allowed]

if unexpected:
    print(f'FAIL:{','.join(unexpected)}')
    sys.exit(1)
else:
    print('PASS')
    sys.exit(0)
")

if [[ "$result" =~ "PASS" ]]; then
    pass "Environment variables minimal (only LC_CTYPE for UTF-8)"
else
    fail "Unexpected environment variables: $result"
fi
echo ""

# Test 6: Time Limit Enforcement
echo "Test 6: Time Limit Enforcement (30s)"
echo "-------------------------------------"
start_time=$(date +%s)
podman run --rm --cap-add=SYS_ADMIN --entrypoint sh pathfinder:sandbox-test -c "
cat > /tmp/test.py <<'PYEOF'
import time
time.sleep(35)
PYEOF

mkdir -p /tmp/nsjail_root
nsjail -Mo --user nobody --chroot /tmp/nsjail_root --iface_no_lo --disable_proc \
  --bindmount_ro /usr:/usr --bindmount_ro /lib:/lib --bindmount /tmp:/tmp --cwd /tmp \
  --rlimit_as 512 --rlimit_cpu 30 --time_limit 30 --quiet \
  -- /usr/bin/python3 /tmp/test.py 2>&1
" &>/dev/null || true
end_time=$(date +%s)
duration=$((end_time - start_time))

if [ $duration -le 33 ]; then
    pass "Time limit enforced (killed within ~30 seconds, actual: ${duration}s)"
else
    fail "Time limit not enforced (ran for ${duration}s)"
fi
echo ""

# Test 7: CPU Limit Enforcement
echo "Test 7: CPU Limit Enforcement (30s CPU time)"
echo "---------------------------------------------"
start_time=$(date +%s)
podman run --rm --cap-add=SYS_ADMIN --entrypoint sh pathfinder:sandbox-test -c "
cat > /tmp/test.py <<'PYEOF'
x = 0
while True:
    x += 1
PYEOF

mkdir -p /tmp/nsjail_root
nsjail -Mo --user nobody --chroot /tmp/nsjail_root --iface_no_lo --disable_proc \
  --bindmount_ro /usr:/usr --bindmount_ro /lib:/lib --bindmount /tmp:/tmp --cwd /tmp \
  --rlimit_as 512 --rlimit_cpu 30 --time_limit 35 --quiet \
  -- /usr/bin/python3 /tmp/test.py 2>&1
" &>/dev/null || true
end_time=$(date +%s)
duration=$((end_time - start_time))

if [ $duration -le 33 ]; then
    pass "CPU limit enforced (killed within ~30s CPU time, actual: ${duration}s)"
else
    fail "CPU limit not enforced (ran for ${duration}s)"
fi
echo ""

# Test 8: Memory Limit Enforcement
echo "Test 8: Memory Limit Enforcement (512MB)"
echo "-----------------------------------------"
result=$(run_test "
import sys
try:
    data = []
    for i in range(600):
        data.append('x' * 1024 * 1024)
    print('FAIL')
    sys.exit(1)
except MemoryError:
    print('PASS')
    sys.exit(0)
" 2>&1 || echo "PASS")

if [[ "$result" =~ "PASS" ]] || [[ "$result" =~ "Killed" ]]; then
    pass "Memory limit enforced (512MB)"
else
    fail "Memory limit not enforced"
fi
echo ""

# Test 9: File Size Limit Enforcement
echo "Test 9: File Size Limit Enforcement (1MB)"
echo "------------------------------------------"
result=$(run_test "
import sys
try:
    with open('/tmp/bigfile.txt', 'w') as f:
        f.write('x' * 2 * 1024 * 1024)
    print('FAIL')
    sys.exit(1)
except OSError:
    print('PASS')
    sys.exit(0)
" 2>&1 || echo "PASS")

if [[ "$result" =~ "PASS" ]] || [[ "$result" =~ "File size limit exceeded" ]]; then
    pass "File size limit enforced (1MB max)"
else
    fail "File size limit not enforced"
fi
echo ""

# Test 10: /tmp Write Access (Should Work)
echo "Test 10: /tmp Write Access (Should Work)"
echo "-----------------------------------------"
result=$(run_test "
import sys
try:
    with open('/tmp/test_file.txt', 'w') as f:
        f.write('test data')
    with open('/tmp/test_file.txt', 'r') as f:
        content = f.read()
    if content == 'test data':
        print('PASS')
        sys.exit(0)
    else:
        print('FAIL')
        sys.exit(1)
except Exception as e:
    print(f'FAIL:{e}')
    sys.exit(1)
")

if [[ "$result" =~ "PASS" ]]; then
    pass "/tmp is writable (expected behavior for output)"
else
    fail "/tmp write access failed: $result"
fi
echo ""

# Summary
echo "=========================================="
echo "Test Summary"
echo "=========================================="
echo -e "Total Tests:  $TOTAL_TESTS"
echo -e "${GREEN}Passed:       $PASS_COUNT${NC}"
echo -e "${RED}Failed:       $FAIL_COUNT${NC}"
echo ""

if [ $FAIL_COUNT -eq 0 ]; then
    echo -e "${GREEN}✅ ALL TESTS PASSED - nsjail security is working correctly!${NC}"
    exit 0
else
    echo -e "${RED}❌ SOME TESTS FAILED - Please review security configuration${NC}"
    exit 1
fi
