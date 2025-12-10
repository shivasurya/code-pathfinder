"""
DOCKER-BP-003: Deprecated MAINTAINER Instruction

Security Impact: INFO
Category: Best Practice

DESCRIPTION:
This rule detects usage of the deprecated MAINTAINER instruction. The MAINTAINER
instruction has been deprecated since Docker 1.13 (January 2017) in favor of using
LABEL instructions with standardized metadata keys. Using deprecated features can
lead to compatibility issues with newer Docker versions and tooling.

WHY MAINTAINER IS DEPRECATED:

1. **Limited Metadata Support**:
   MAINTAINER only stores a single email/name field, while modern container images
   need rich metadata like:
   - Author contact information
   - Source repository URLs
   - License information
   - Build timestamps
   - Version information

2. **No Standardization**:
   Different organizations used MAINTAINER in inconsistent formats:
   ```dockerfile
   MAINTAINER John Doe
   MAINTAINER john@example.com
   MAINTAINER "John Doe <john@example.com>"
   ```

3. **Future Removal**:
   While currently still supported, deprecated features may be removed in future
   Docker versions, breaking your builds.

4. **Tooling Incompatibility**:
   Modern container scanning and management tools expect OCI-compliant LABEL
   metadata, not MAINTAINER fields.

VULNERABLE EXAMPLE:
```dockerfile
FROM ubuntu:22.04

# Deprecated: Old-style maintainer
MAINTAINER John Doe <john@example.com>

RUN apt-get update && apt-get install -y nginx
CMD ["nginx", "-g", "daemon off;"]
```

SECURE EXAMPLE - Using LABEL:
```dockerfile
FROM ubuntu:22.04

# Modern: Use OCI-compliant labels
LABEL org.opencontainers.image.authors="John Doe <john@example.com>"
LABEL org.opencontainers.image.vendor="ACME Corporation"
LABEL org.opencontainers.image.title="NGINX Web Server"
LABEL org.opencontainers.image.description="Production NGINX server with security hardening"
LABEL org.opencontainers.image.version="1.25.3"
LABEL org.opencontainers.image.url="https://github.com/acme/nginx-image"
LABEL org.opencontainers.image.source="https://github.com/acme/nginx-image"
LABEL org.opencontainers.image.documentation="https://docs.acme.com/nginx"
LABEL org.opencontainers.image.licenses="Apache-2.0"
LABEL org.opencontainers.image.created="2024-01-15T10:30:00Z"

RUN apt-get update && apt-get install -y nginx
CMD ["nginx", "-g", "daemon off;"]
```

STANDARD LABEL KEYS (OCI Image Spec):

The Open Container Initiative (OCI) defines standard label keys:

```dockerfile
# Required for image identification
LABEL org.opencontainers.image.created="2024-01-15T10:30:00Z"
LABEL org.opencontainers.image.authors="Team Name <team@example.com>"
LABEL org.opencontainers.image.url="https://github.com/org/repo"
LABEL org.opencontainers.image.documentation="https://docs.example.com"
LABEL org.opencontainers.image.source="https://github.com/org/repo"
LABEL org.opencontainers.image.version="1.2.3"
LABEL org.opencontainers.image.vendor="Company Name"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.title="Application Name"
LABEL org.opencontainers.image.description="Brief description"

# Optional for Git integration
LABEL org.opencontainers.image.revision="abc123def456"
LABEL org.opencontainers.image.ref.name="v1.2.3"
```

AUTOMATED LABEL INJECTION:

For CI/CD pipelines, inject build metadata automatically:

```dockerfile
# Use ARG for build-time variables
ARG BUILD_DATE
ARG VCS_REF
ARG VERSION

LABEL org.opencontainers.image.created="${BUILD_DATE}"
LABEL org.opencontainers.image.revision="${VCS_REF}"
LABEL org.opencontainers.image.version="${VERSION}"
```

Build command:
```bash
docker build \
  --build-arg BUILD_DATE=$(date -u +'%Y-%m-%dT%H:%M:%SZ') \
  --build-arg VCS_REF=$(git rev-parse --short HEAD) \
  --build-arg VERSION=$(git describe --tags) \
  -t myapp:latest .
```

ADDITIONAL USEFUL LABELS:

```dockerfile
# Custom organizational labels
LABEL com.example.team="platform-engineering"
LABEL com.example.environment="production"
LABEL com.example.tier="frontend"
LABEL com.example.cost-center="engineering"

# Security and compliance
LABEL com.example.security.scan-date="2024-01-15"
LABEL com.example.security.scanner="trivy-v0.48"
LABEL com.example.compliance.framework="SOC2"

# Operational metadata
LABEL com.example.monitoring.metrics-port="9090"
LABEL com.example.monitoring.health-endpoint="/health"
LABEL com.example.backup.schedule="0 2 * * *"
```

QUERYING LABELS:

Labels are searchable and can be used for automation:

```bash
# Inspect all labels
docker inspect myapp:latest | jq '.[0].Config.Labels'

# Filter containers by label
docker ps --filter "label=com.example.team=platform-engineering"

# Find images by version
docker images --filter "label=org.opencontainers.image.version=1.2.3"
```

Kubernetes integration:
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: myapp
spec:
  containers:
  - name: app
    image: myapp:latest
    env:
    # Extract version from image label
    - name: APP_VERSION
      valueFrom:
        fieldRef:
          fieldPath: metadata.labels['org.opencontainers.image.version']
```

MIGRATION GUIDE:

**Step 1: Extract current MAINTAINER value**
```dockerfile
# Old Dockerfile
MAINTAINER John Doe <john@example.com>
```

**Step 2: Convert to LABEL**
```dockerfile
# New Dockerfile
LABEL org.opencontainers.image.authors="John Doe <john@example.com>"
```

**Step 3: Add additional metadata**
```dockerfile
LABEL org.opencontainers.image.authors="John Doe <john@example.com>"
LABEL org.opencontainers.image.url="https://github.com/myorg/myrepo"
LABEL org.opencontainers.image.documentation="https://docs.myorg.com"
LABEL org.opencontainers.image.source="https://github.com/myorg/myrepo"
LABEL org.opencontainers.image.vendor="My Organization"
LABEL org.opencontainers.image.licenses="Apache-2.0"
```

**Step 4: Consolidate multiple labels (optional)**
```dockerfile
# Can combine related labels on one line
LABEL org.opencontainers.image.authors="John Doe <john@example.com>" \
      org.opencontainers.image.vendor="My Organization" \
      org.opencontainers.image.licenses="Apache-2.0"
```

BEST PRACTICES:

1. **Use OCI Standard Keys**: Stick to `org.opencontainers.image.*` for portability
2. **Namespace Custom Labels**: Use reverse DNS notation (com.yourcompany.*)
3. **Document Label Schema**: Maintain a list of labels your organization uses
4. **Automate Injection**: Use build args and CI/CD variables
5. **Version Labels**: Always include version information for tracking
6. **Audit Labels**: Include security scanning and compliance metadata

REMEDIATION:
Replace MAINTAINER with appropriate LABEL instructions:

```dockerfile
# Before (deprecated)
MAINTAINER ops-team@example.com

# After (modern)
LABEL org.opencontainers.image.authors="ops-team@example.com"
LABEL org.opencontainers.image.vendor="Example Corp"
```

REFERENCES:
- OCI Image Specification - Annotations
- Docker LABEL documentation
- Docker Deprecated Features
- Label Schema Convention (historical reference)
"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction


@dockerfile_rule(
    id="DOCKER-BP-003",
    name="Deprecated MAINTAINER Instruction",
    severity="INFO",
    category="best-practice",
    message="MAINTAINER instruction is deprecated. Use LABEL org.opencontainers.image.authors instead."
)
def maintainer_deprecated():
    """
    Detects usage of deprecated MAINTAINER instruction.

    The MAINTAINER instruction is deprecated since Docker 1.13 in favor
    of LABEL instructions with standardized OCI metadata keys.
    """
    return instruction(type="MAINTAINER")
