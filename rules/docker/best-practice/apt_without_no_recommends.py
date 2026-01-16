"""
DOCKER-BP-005: apt-get Without --no-install-recommends

Security Impact: LOW
Category: Best Practice

DESCRIPTION:
This rule detects RUN instructions that use `apt-get install` without the
`--no-install-recommends` flag. By default, apt installs both required packages
and "recommended" packages, which are often unnecessary and significantly bloat
Docker images, increasing attack surface and build/deploy times.

IMAGE BLOAT IMPACT:

Without --no-install-recommends, a simple package installation can balloon:

```dockerfile
# Example: Installing nginx WITHOUT --no-install-recommends
FROM ubuntu:22.04
RUN apt-get update && apt-get install -y nginx
# Result: 265 MB image
# Installs: nginx + 50+ recommended packages
```

```dockerfile
# Example: Installing nginx WITH --no-install-recommends
FROM ubuntu:22.04
RUN apt-get update && apt-get install -y --no-install-recommends nginx
# Result: 185 MB image
# Installs: Only nginx + essential dependencies
# Savings: ~80 MB (30% reduction)
```

Real-world impact on common packages:
- nginx: 265 MB → 185 MB (30% reduction)
- python3: 420 MB → 280 MB (33% reduction)
- postgresql-client: 340 MB → 210 MB (38% reduction)
- curl: 160 MB → 90 MB (44% reduction)

SECURITY IMPLICATIONS:

1. **Increased Attack Surface**:
   Every additional package is potential vulnerability:
   - More binaries that could have security flaws
   - More libraries with CVEs
   - More services that could be exploited

2. **Supply Chain Risks**:
   Recommended packages may pull in:
   - Unmaintained dependencies
   - Packages from less-trusted sources
   - Transitive dependencies with vulnerabilities

3. **Compliance Issues**:
   - More packages = more licenses to track
   - Harder to maintain Software Bill of Materials (SBOM)
   - Difficult to audit all dependencies

VULNERABLE EXAMPLE:
```dockerfile
FROM ubuntu:22.04

# Bad: Installs nginx + 50+ recommended packages
RUN apt-get update && \
    apt-get install -y nginx curl

# This pulls in:
# - X11 libraries (for desktop apps, not needed in containers)
# - Documentation packages
# - Example/demo files
# - Optional utilities
```

What gets installed unnecessarily:
```
nginx-core
  + nginx-common
  + libgd3
    + fonts-dejavu-core (not needed)
    + fontconfig-config (not needed)
  + libxpm4 (X11 library, not needed)
  + libx11-6 (X11 library, not needed)
  + perl-modules (may not be needed)
  + ... 40+ more packages
```

SECURE EXAMPLE:
```dockerfile
FROM ubuntu:22.04

# Good: Only install required packages
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
      nginx \
      curl \
      ca-certificates && \
    # Clean up apt cache to further reduce size
    rm -rf /var/lib/apt/lists/*

# Result: Smaller, more secure image
```

COMPLETE BEST PRACTICE PATTERN:

```dockerfile
FROM ubuntu:22.04

# Combine update + install + cleanup in single RUN to minimize layers
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
      # List packages explicitly for documentation
      nginx=1.18.0-6ubuntu14.4 \  # Pin versions for reproducibility
      curl=7.81.0-1ubuntu1.15 \
      ca-certificates \
      tzdata && \
    # Remove apt cache (saves ~20-40 MB)
    rm -rf /var/lib/apt/lists/* && \
    # Remove unnecessary files
    rm -rf /usr/share/doc/* && \
    rm -rf /usr/share/man/* && \
    rm -rf /var/cache/apt/*

# Total cleanup can save 50-100 MB
```

UNDERSTANDING APT PACKAGE TYPES:

**Depends**: Required dependencies (always installed)
```
Package: nginx
Depends: nginx-core, lsb-base
```

**Recommends**: Strongly suggested but not required (installed by default)
```
Package: nginx
Recommends: nginx-doc, fonts-dejavu-core
```

**Suggests**: Optional enhancements (not installed by default)
```
Package: nginx
Suggests: nginx-module-*
```

With --no-install-recommends, only "Depends" are installed.

WHEN YOU MIGHT NEED RECOMMENDED PACKAGES:

1. **Desktop Applications** (rare in containers):
   If you're containerizing a GUI app that needs X11 libraries

2. **Development Images**:
   Development containers may benefit from additional tools:
   ```dockerfile
   # development.Dockerfile
   RUN apt-get install -y curl  # Allow recommends for dev tools
   ```

3. **Specific Functionality**:
   Some packages need recommended dependencies:
   ```dockerfile
   # If you specifically need the docs
   RUN apt-get install -y nginx nginx-doc
   ```

ADVANCED OPTIMIZATION TECHNIQUES:

**1. Multi-Stage Builds to Isolate Build Dependencies**:
```dockerfile
# Build stage with full tools
FROM ubuntu:22.04 AS builder
RUN apt-get update && apt-get install -y \
    build-essential \
    git
COPY . /app
RUN cd /app && make build

# Runtime stage with minimal packages
FROM ubuntu:22.04
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates && \
    rm -rf /var/lib/apt/lists/*
COPY --from=builder /app/dist /app
```

**2. Use Minimal Base Images**:
```dockerfile
# Instead of ubuntu:22.04 (77 MB)
FROM ubuntu:22.04

# Consider debian:bookworm-slim (74 MB)
FROM debian:bookworm-slim

# Or alpine:3.19 (7 MB)
FROM alpine:3.19
# Alpine uses apk, not apt, but same principle applies
```

**3. Verify Installed Packages**:
```bash
# List all installed packages in your image
docker run --rm myimage dpkg -l

# Compare image sizes
docker images myimage

# Analyze image layers
docker history myimage
```

REMEDIATION STEPS:

**Step 1: Add the flag**
```dockerfile
# Before
RUN apt-get update && apt-get install -y curl

# After
RUN apt-get update && apt-get install -y --no-install-recommends curl
```

**Step 2: Test your application**
Ensure functionality isn't broken by missing recommended packages

**Step 3: Add cleanup**
```dockerfile
RUN apt-get update && \
    apt-get install -y --no-install-recommends curl && \
    rm -rf /var/lib/apt/lists/*
```

**Step 4: Verify size reduction**
```bash
docker build -t myapp:before --target before .
docker build -t myapp:after --target after .
docker images | grep myapp
```

AUTOMATION AND ENFORCEMENT:

**Dockerfile Linter (hadolint)**:
```bash
hadolint Dockerfile
# DL3008: Pin versions in apt-get install
# DL3009: Delete apt cache after installing
# DL3015: Use --no-install-recommends
```

**CI/CD Integration**:
```yaml
# .github/workflows/docker.yml
- name: Lint Dockerfile
  run: |
    docker run --rm -i hadolint/hadolint < Dockerfile
```

REFERENCES:
- Debian Policy Manual: Dependencies
- Docker Best Practices: Minimize Layer Size
- Ubuntu Package Management Guide
- CIS Docker Benchmark: Image Optimization
"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction
from rules.container_combinators import all_of


@dockerfile_rule(
    id="DOCKER-BP-005",
    name="apt-get Without --no-install-recommends",
    severity="LOW",
    cwe="CWE-710",
    category="best-practice",
    tags="docker,dockerfile,apt-get,package-manager,ubuntu,debian,optimization,image-size,best-practice,bloat,attack-surface",
    message="apt-get install without --no-install-recommends. This installs unnecessary packages, increasing image size and attack surface."
)
def apt_without_no_recommends():
    """
    Detects apt-get install without --no-install-recommends flag.

    Without this flag, apt installs "recommended" packages which are
    often not needed and bloat the image by 30-50%.
    """
    return all_of(
        instruction(type="RUN", contains="apt-get install"),
        instruction(type="RUN", not_contains="--no-install-recommends")
    )
