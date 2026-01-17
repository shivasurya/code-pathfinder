"""
DOCKER-AUD-001: Dockerfile Source Not Pinned

Security Impact: LOW
Audit Issue

DESCRIPTION:
Detects FROM instructions without digest pinning (@sha256:...).
Tags are mutable and can be updated to point to different images, while
digests are immutable references to specific image layers.

WHY THIS IS PROBLEMATIC:
1. Mutable Tags: Tags like 'latest' or '1.24' can be updated
2. Non-Reproducible: Same tag may produce different images over time
3. Supply Chain Risk: Tag could be compromised without detection
4. Build Drift: CI/CD builds may produce inconsistent results
5. Security Updates: Image may change unexpectedly

VULNERABLE EXAMPLE:
```dockerfile
# Bad: Tag can be updated to point to different image
FROM nginx:1.24.0
FROM ubuntu:latest  # Especially bad - very mutable

# These can all change over time
FROM node:18
FROM python:3.11-slim
```

SECURE EXAMPLE:
```dockerfile
# Good: Digest pinning ensures immutable reference
FROM nginx:1.24.0@sha256:a4f34e6fb432af40bc594a0f1e5178598f6ca0f1ea6b3c6e7c5e5e8c3f9d6e1a

# Can combine readable tag with immutable digest
FROM ubuntu:22.04@sha256:b6b83d3c331794420340093eb706a6f152d9c1fa51b262d9bf34594887c2c7ac

# For multi-stage builds
FROM node:18@sha256:abc123... AS builder
FROM nginx:alpine@sha256:def456... AS runtime
```

REMEDIATION:
1. Find the digest for your image:
   ```bash
   docker pull nginx:1.24.0
   docker inspect nginx:1.24.0 | grep RepoDigests
   ```

2. Update Dockerfile with digest:
   ```dockerfile
   FROM nginx:1.24.0@sha256:...
   ```

3. For automated updates, use tools like:
   - Renovate Bot
   - Dependabot
   - Docker Hub webhooks

Note: This is an audit check. Digest pinning increases reproducibility but
requires more maintenance. Evaluate based on your security requirements.

REFERENCES:
- Docker Content Trust
- hadolint DL3026
- Supply Chain Security Best Practices
"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction


@dockerfile_rule(
    id="DOCKER-AUD-001",
    name="Dockerfile Source Not Pinned",
    severity="LOW",
    cwe="CWE-1188",
    category="audit",
    tags="docker,dockerfile,from,digest,sha256,immutability,supply-chain,reproducibility,audit,security",
    message="FROM instruction without digest pinning. Consider using @sha256:... for immutable builds."
)
def dockerfile_source_not_pinned():
    return instruction(type="FROM", missing_digest=True)
