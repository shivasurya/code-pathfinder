"""
DOCKER-BP-007: apk add Without --no-cache

Security Impact: LOW
Category: Best Practice

DESCRIPTION:
This rule detects RUN instructions using Alpine Linux's `apk add` command without
the `--no-cache` flag. Alpine's package manager caches downloaded packages in
`/var/cache/apk/`, which unnecessarily increases Docker image size. The --no-cache
flag prevents caching, keeping images minimal.

ALPINE LINUX PACKAGE CACHING:

Alpine Linux (apk) behaves differently from Debian/Ubuntu (apt):

**Default apt behavior (Debian/Ubuntu)**:
```bash
apt-get update          # Updates package lists
apt-get install curl    # Downloads and installs
# Leaves cache in /var/cache/apt/archives/
```

**Default apk behavior (Alpine)**:
```bash
apk update              # Updates package index
apk add curl            # Downloads and installs
# Leaves cache in /var/cache/apk/
```

The key difference: Alpine's --no-cache flag combines update + install + cleanup:
```bash
apk add --no-cache curl
# Equivalent to:
# apk update
# apk add curl
# rm -rf /var/cache/apk/*
```

IMAGE SIZE IMPACT:

```dockerfile
# Without --no-cache
FROM alpine:3.19
RUN apk add curl
# Result: 12.5 MB image
# Cache retained: /var/cache/apk/* (~2-3 MB)
```

```dockerfile
# With --no-cache
FROM alpine:3.19
RUN apk add --no-cache curl
# Result: 9.8 MB image
# Savings: ~2.7 MB (22% reduction)
```

Real-world examples:
- nginx: 15 MB → 12 MB (20% reduction)
- python3: 55 MB → 52 MB (5% reduction)
- postgresql-client: 25 MB → 22 MB (12% reduction)

VULNERABLE EXAMPLE:
```dockerfile
FROM alpine:3.19

# Bad: Leaves package cache in image
RUN apk update
RUN apk add \
    nginx \
    curl \
    ca-certificates

# Cache files remain in /var/cache/apk/*
# Adds 2-5 MB of unnecessary data
```

Why this is problematic:
1. Wasted space in every layer
2. Slower image pulls/pushes
3. More storage costs in registries
4. Longer container startup times

SECURE EXAMPLE:
```dockerfile
FROM alpine:3.19

# Good: No cache retained
RUN apk add --no-cache \
    nginx \
    curl \
    ca-certificates

# Even better: Pin versions
RUN apk add --no-cache \
    nginx=1.24.0-r15 \
    curl=8.4.0-r0 \
    ca-certificates=20230506-r0
```

ALPINE PACKAGE MANAGEMENT BEST PRACTICES:

**1. Always Use --no-cache**:
```dockerfile
# Single package
RUN apk add --no-cache curl

# Multiple packages
RUN apk add --no-cache \
    package1 \
    package2 \
    package3
```

**2. Combine Operations in Single RUN**:
```dockerfile
# Bad: Multiple layers with cache
RUN apk add --no-cache nginx
RUN apk add --no-cache curl
# Creates 2 layers

# Good: Single layer
RUN apk add --no-cache \
    nginx \
    curl
# Creates 1 layer, more efficient
```

**3. Pin Package Versions**:
```dockerfile
RUN apk add --no-cache \
    python3=3.11.6-r0 \
    py3-pip=23.3.1-r0
```

Find available versions:
```bash
docker run --rm alpine:3.19 apk search -e python3
# python3-3.11.6-r0
```

**4. Virtual Packages for Build Dependencies**:
```dockerfile
# Install build deps, compile, then remove in same layer
RUN apk add --no-cache --virtual .build-deps \
        gcc \
        musl-dev \
        libffi-dev && \
    pip install cryptography && \
    # Remove build deps
    apk del .build-deps

# Build tools don't remain in final image
```

**5. Update + Install Pattern**:
```dockerfile
# If you need latest package index
RUN apk update && apk add --no-cache curl
# Or use --no-cache which updates automatically
RUN apk add --no-cache curl
```

COMPARISON WITH DEBIAN/UBUNTU PATTERN:

**Alpine (apk)**:
```dockerfile
FROM alpine:3.19
RUN apk add --no-cache nginx
```

**Debian (apt)**:
```dockerfile
FROM debian:bookworm-slim
RUN apt-get update && \
    apt-get install -y --no-install-recommends nginx && \
    rm -rf /var/lib/apt/lists/*
```

**Ubuntu (apt)**:
```dockerfile
FROM ubuntu:22.04
RUN apt-get update && \
    apt-get install -y --no-install-recommends nginx && \
    rm -rf /var/lib/apt/lists/*
```

Alpine is more concise due to --no-cache handling everything.

ADVANCED ALPINE TECHNIQUES:

**1. Virtual Build Dependencies**:
```dockerfile
FROM alpine:3.19

# Install build deps with virtual package
RUN apk add --no-cache --virtual .build-deps \
        build-base \
        python3-dev \
        libffi-dev \
        openssl-dev && \
    \
    # Build application
    pip install --no-cache-dir cryptography && \
    \
    # Remove all build deps in one command
    apk del .build-deps

# Result: Only runtime deps remain
```

**2. Edge Repository for Latest Packages**:
```dockerfile
FROM alpine:3.19

# Add edge repository for bleeding-edge packages
RUN apk add --no-cache \
    --repository=http://dl-cdn.alpinelinux.org/alpine/edge/main \
    --repository=http://dl-cdn.alpinelinux.org/alpine/edge/community \
    nginx
```

**3. Testing Packages**:
```dockerfile
FROM alpine:3.19

# Use testing repository
RUN apk add --no-cache \
    --repository=http://dl-cdn.alpinelinux.org/alpine/edge/testing \
    some-new-package
```

**4. Multi-Architecture Support**:
```dockerfile
FROM alpine:3.19

# Works on amd64, arm64, armv7, etc.
RUN apk add --no-cache curl
# Alpine's apk handles architecture automatically
```

WHEN TO USE ALPINE:

**Good Use Cases**:
- Microservices with minimal dependencies
- Simple HTTP servers (nginx, caddy)
- CLI tools and utilities
- Lambda functions
- Static binaries (Go, Rust)

**Avoid Alpine When**:
- Using glibc-dependent binaries (Alpine uses musl libc)
- Need extensive system libraries
- Python packages with C extensions may have compatibility issues
- Java applications (JVM overhead negates size benefit)

ALPINE ALTERNATIVES:

If Alpine causes compatibility issues:

```dockerfile
# Distroless (Google)
FROM gcr.io/distroless/static-debian12
# 2 MB, no package manager, maximum security

# Chainguard
FROM cgr.dev/chainguard/static
# Ultra-minimal, SBOMs included

# Debian Slim
FROM debian:bookworm-slim
# 74 MB, more compatible than Alpine

# Ubuntu Minimal
FROM ubuntu:22.04-minimal
# 29 MB, better compatibility
```

DEBUGGING TIPS:

**Check package cache size**:
```bash
docker run --rm myimage du -sh /var/cache/apk
```

**List installed packages**:
```bash
docker run --rm myimage apk info
```

**Verify package versions**:
```bash
docker run --rm myimage apk info nginx
```

**Compare image sizes**:
```bash
docker history myimage
docker images myimage
```

REMEDIATION:

**Before (with cache)**:
```dockerfile
RUN apk update
RUN apk add nginx curl
```

**After (no cache)**:
```dockerfile
RUN apk add --no-cache \
    nginx \
    curl
```

**Advanced (with version pinning)**:
```dockerfile
RUN apk add --no-cache \
    nginx=1.24.0-r15 \
    curl=8.4.0-r0 \
    ca-certificates=20230506-r0
```

AUTOMATION:

**Hadolint Detection**:
```bash
hadolint Dockerfile
# DL3018: Pin versions in apk add
# DL3019: Use --no-cache with apk add
```

**CI/CD Check**:
```yaml
- name: Verify Alpine best practices
  run: |
    grep -q "apk add --no-cache" Dockerfile || exit 1
```

REFERENCES:
- Alpine Linux Package Management
- Docker Official Images: Alpine Best Practices
- Alpine Linux Wiki: Package Management
- Docker Multi-Stage Builds with Alpine
"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction
from rules.container_combinators import all_of


@dockerfile_rule(
    id="DOCKER-BP-007",
    name="apk add Without --no-cache",
    severity="LOW",
    category="best-practice",
    message="apk add without --no-cache. Package cache remains in image, increasing size by 2-5 MB."
)
def apk_without_no_cache():
    """
    Detects apk add without --no-cache flag for Alpine images.

    The --no-cache flag prevents package cache from being stored
    in the image, reducing size by 20-30% for Alpine-based images.
    """
    return all_of(
        instruction(type="RUN", contains="apk add"),
        instruction(type="RUN", not_contains="--no-cache")
    )
