#!/bin/bash
set -e

echo "=== Testing nsjail Installation in Chainguard Wolfi Base ==="

# Create test Dockerfile
cat > /tmp/test-nsjail.Dockerfile <<'EOF'
FROM cgr.dev/chainguard/wolfi-base:latest

# Try Option 1: Alpine edge apk
RUN echo "@edge https://dl-cdn.alpinelinux.org/alpine/edge/community" >> /etc/apk/repositories && \
    apk add --no-cache nsjail@edge || echo "APK install failed, will try build from source"

# Fallback to build from source if needed
RUN if ! command -v nsjail &> /dev/null; then \
        apk add --no-cache build-base protobuf-dev libnl3-dev git flex bison && \
        git clone --depth 1 https://github.com/google/nsjail.git /tmp/nsjail && \
        cd /tmp/nsjail && \
        sed -i 's/-Werror//g' Makefile && \
        make && \
        cp nsjail /usr/bin/ && \
        cd / && rm -rf /tmp/nsjail && \
        apk del build-base git flex bison; \
    fi

# Install Python 3.14
RUN apk add --no-cache python3 py3-pip

CMD ["/bin/sh"]
EOF

# Build test image
echo "Building test image..."
podman build -f /tmp/test-nsjail.Dockerfile -t test-nsjail:latest .

# Test 1: Verify nsjail is installed
echo ""
echo "Test 1: Checking nsjail installation..."
podman run --rm test-nsjail:latest nsjail --version

# Test 2: Verify Python is installed
echo ""
echo "Test 2: Checking Python installation..."
podman run --rm test-nsjail:latest python3 --version

# Test 3: Run simple nsjail command
echo ""
echo "Test 3: Running simple nsjail test..."
podman run --rm test-nsjail:latest nsjail \
    --mode l \
    --user nobody \
    --chroot / \
    --disable_proc \
    -- /usr/bin/python3 --version

# Test 4: Test Python script execution in nsjail
echo ""
echo "Test 4: Running Python script in nsjail..."
podman run --rm test-nsjail:latest /bin/sh -c '
cat > /tmp/test.py <<PYEOF
print("Hello from sandboxed Python!")
import sys
print(f"Python version: {sys.version}")
PYEOF

nsjail \
    --mode l \
    --user nobody \
    --chroot / \
    --disable_proc \
    -- /usr/bin/python3 /tmp/test.py
'

echo ""
echo "=== All tests passed! ==="
echo "nsjail installation method: (check output above)"
