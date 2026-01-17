"""
DOCKER-BP-009: Avoid dnf update in Dockerfile

Security Impact: MEDIUM
Best Practice Violation

DESCRIPTION:
Detects use of 'dnf update' in Dockerfiles. Running system updates in Docker builds
creates unpredictable, non-reproducible images and can introduce breaking changes or
security vulnerabilities.

WHY THIS IS PROBLEMATIC:
1. Non-Reproducible Builds: Different build times produce different images
2. Breaks Caching: Forces rebuild of all subsequent layers
3. Unpredictable Versions: Can install untested package versions
4. Security Risk: May introduce vulnerabilities or break dependencies

VULNERABLE EXAMPLE:
```dockerfile
FROM fedora:latest
RUN dnf update -y  # Bad: Unpredictable, non-reproducible builds
RUN dnf install -y nginx
```

SECURE EXAMPLE:
```dockerfile
FROM fedora:38  # Specific version
RUN dnf install -y nginx-1.24.0-1.fc38
# Or install latest from specific base image
RUN dnf install -y nginx
```

REMEDIATION:
1. Remove 'dnf update' commands
2. Use specific base image versions (e.g., fedora:38 instead of fedora:latest)
3. Install specific package versions when needed
4. Rebuild images periodically with updated base images

REFERENCES:
- Docker Best Practices
- hadolint DL3039
"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction


@dockerfile_rule(
    id="DOCKER-BP-009",
    name="Avoid dnf update",
    severity="MEDIUM",
    cwe="CWE-710",
    category="best-practice",
    tags="docker,dockerfile,dnf,package-manager,fedora,rhel,update,reproducibility,best-practice,anti-pattern",
    message="Avoid 'dnf update' in Dockerfiles. Use specific base image versions for reproducible builds."
)
def avoid_dnf_update():
    return instruction(type="RUN", contains="dnf update")
