"""
DOCKER-BP-001: Base Image Uses :latest Tag

Security Impact: MEDIUM
Category: Best Practice

DESCRIPTION:
This rule detects FROM instructions using the :latest tag (or no tag, which defaults
to :latest). Using unversioned or latest-tagged base images makes builds non-reproducible
and can introduce unexpected breaking changes or security vulnerabilities when the
upstream image is updated.

SECURITY AND RELIABILITY IMPLICATIONS:

1. **Non-Reproducible Builds**:
   Building the same Dockerfile at different times can produce different images if the
   upstream :latest tag has been updated. This violates the principle of immutable
   infrastructure and makes it impossible to reliably recreate previous builds.

2. **Untested Updates**:
   The :latest tag may include:
   - Breaking API changes
   - New security vulnerabilities
   - Incompatible library versions
   - Modified system configurations
   These changes bypass your testing and QA processes.

3. **Supply Chain Attacks**:
   If an attacker compromises the upstream image repository, they can push malicious
   code to the :latest tag, affecting all builds that pull it. Version pinning limits
   the window of opportunity for such attacks.

4. **Debugging Nightmares**:
   When a production issue occurs, you cannot determine which version of the base
   image was used, making root cause analysis extremely difficult.

5. **Inconsistent Environments**:
   Development, staging, and production may end up running different base image versions,
   leading to "works on my machine" problems at scale.

VULNERABLE EXAMPLE:
```dockerfile
# Bad: Uses implicit :latest tag
FROM ubuntu

# Bad: Explicit :latest tag
FROM python:latest

# Bad: Missing digest verification
FROM node:18-alpine

# Multiple stages with :latest
FROM maven:latest AS builder
FROM openjdk:latest
```

What happens in practice:
```bash
# Day 1: Build image
docker build -t myapp:v1.0 .
# Uses ubuntu:22.04 (current :latest)

# Day 30: Rebuild same Dockerfile
docker build -t myapp:v1.0 .
# Uses ubuntu:24.04 (new :latest)
# Same tag, different content!
```

SECURE EXAMPLE:
```dockerfile
# Good: Pin specific version
FROM ubuntu:22.04

# Better: Use digest for cryptographic verification
FROM ubuntu:22.04@sha256:b6b83d3c331794420340093eb706a6f152d9c1fa51b262d9bf34594887c2c7ac

# Good: Pin Python minor version
FROM python:3.11.7-slim-bookworm

# Good: Pin Node.js with specific base
FROM node:18.19.0-alpine3.19

# Multi-stage with pinned versions
FROM maven:3.9.6-eclipse-temurin-17 AS builder
FROM eclipse-temurin:17.0.10_7-jre-alpine
```

CHOOSING THE RIGHT VERSION TAG:

**1. Semantic Versioning Tags:**
```dockerfile
# Least specific (not recommended)
FROM python:3

# Better - minor version pinned
FROM python:3.11

# Best - patch version pinned
FROM python:3.11.7

# Production - add base OS variant
FROM python:3.11.7-slim-bookworm
```

**2. Digest Pinning (Most Secure):**
```dockerfile
# Immutable reference by content hash
FROM python:3.11.7-slim@sha256:abc123def456...

# Find digest:
# docker pull python:3.11.7-slim
# docker inspect python:3.11.7-slim | grep -A 1 RepoDigests
```

**3. Base OS Versioning:**
```dockerfile
# Specify exact OS version
FROM node:18-alpine3.19      # Alpine Linux 3.19
FROM python:3.11-bookworm    # Debian Bookworm
FROM ubuntu:22.04            # Ubuntu 22.04 LTS
```

BEST PRACTICES:

1. **Pin Major + Minor Version Minimum**:
   ```dockerfile
   FROM postgres:15.4  # Not postgres:15 or postgres:latest
   ```

2. **Use Digest Pinning for Production**:
   ```dockerfile
   FROM nginx:1.25.3@sha256:4c0fdaa8b6341bfdeca5f18f7837462c80cff90527ee35ef185571e1c327beac
   ```

3. **Document Version Selection**:
   ```dockerfile
   # Python 3.11.7 selected for security fixes in 3.11.6
   # Base: slim-bookworm for smaller size vs full Debian
   FROM python:3.11.7-slim-bookworm@sha256:abc123...
   ```

4. **Automate Dependency Updates**:
   - Use Dependabot or Renovate to create PRs for base image updates
   - Test updates in CI/CD before merging
   - Review changelogs for breaking changes

5. **Use Minimal Base Images**:
   ```dockerfile
   # Preferred order (smallest to largest):
   FROM scratch                    # Empty base
   FROM gcr.io/distroless/static  # Google distroless
   FROM alpine:3.19               # Alpine Linux
   FROM debian:bookworm-slim      # Debian slim
   FROM ubuntu:22.04              # Full Ubuntu (avoid if possible)
   ```

6. **Scan Images Regularly**:
   Even pinned versions can have vulnerabilities discovered after release.
   ```bash
   docker scan myapp:latest
   trivy image myapp:latest
   ```

REMEDIATION WORKFLOW:

**Step 1: Identify Current Version**
```bash
# Find the exact version your build is using
docker pull ubuntu:latest
docker inspect ubuntu:latest | grep -E "DISTRIB_RELEASE|VERSION"
```

**Step 2: Update Dockerfile**
```dockerfile
# Before
FROM ubuntu

# After
FROM ubuntu:22.04@sha256:b6b83d3c331794420340093eb706a6f152d9c1fa51b262d9bf34594887c2c7ac
```

**Step 3: Test Thoroughly**
```bash
docker build -t myapp:test .
docker run --rm myapp:test pytest
```

**Step 4: Implement Monitoring**
Set up automated scanning and update notifications for pinned versions.

EXCEPTIONS:
- Local development Dockerfiles (docker-compose.dev.yml) may use :latest for convenience
- CI/CD builder images where reproducibility is less critical
- Internal base images maintained by your team with strict version control

REFERENCES:
- Docker Best Practices: Image Tagging
- 12-Factor App: Dependencies
- NIST SP 800-190: Container Image Integrity
- CIS Docker Benchmark: Section 4.1
"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction


@dockerfile_rule(
    id="DOCKER-BP-001",
    name="Base Image Uses :latest Tag",
    severity="MEDIUM",
    cwe="CWE-1188",
    category="best-practice",
    tags="docker,dockerfile,from,image,tag,version,latest,reproducibility,best-practice,supply-chain,immutability",
    message="Base image uses ':latest' tag or no tag (defaults to latest). This makes builds non-reproducible."
)
def using_latest_tag():
    """
    Detects FROM instructions using :latest or implicit latest tag.

    Using :latest leads to non-reproducible builds as the underlying
    image can change at any time. Always pin to specific versions.
    """
    return instruction(type="FROM", image_tag="latest")
