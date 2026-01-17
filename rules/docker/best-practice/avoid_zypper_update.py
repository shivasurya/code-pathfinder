"""
DOCKER-BP-019: Avoid zypper update in Dockerfile

Security Impact: MEDIUM
Best Practice Violation

DESCRIPTION:
Detects use of 'zypper update' in Dockerfiles. Running system updates in Docker builds
creates unpredictable, non-reproducible images and can introduce breaking changes or
security vulnerabilities.

WHY THIS IS PROBLEMATIC:
1. Non-Reproducible Builds: Different build times produce different images
2. Breaks Caching: Forces rebuild of all subsequent layers
3. Unpredictable Versions: Can install untested package versions
4. Security Risk: May introduce vulnerabilities or break dependencies

VULNERABLE EXAMPLE:
```dockerfile
FROM opensuse/leap:latest
RUN zypper update -y  # Bad: Unpredictable, non-reproducible builds
RUN zypper install -y nginx
```

SECURE EXAMPLE:
```dockerfile
FROM opensuse/leap:15.4  # Specific version
RUN zypper install -y nginx-1.21.6-1
# Or install latest from specific base image
RUN zypper install -y nginx
```

REMEDIATION:
1. Remove 'zypper update' commands
2. Use specific base image versions (e.g., opensuse/leap:15.4 instead of :latest)
3. Install specific package versions when needed
4. Rebuild images periodically with updated base images

REFERENCES:
- Docker Best Practices
- hadolint DL3035
"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction


@dockerfile_rule(
    id="DOCKER-BP-019",
    name="Avoid zypper update",
    severity="MEDIUM",
    cwe="CWE-710",
    category="best-practice",
    tags="docker,dockerfile,zypper,package-manager,opensuse,suse,update,reproducibility,best-practice,anti-pattern",
    message="Avoid 'zypper update' in Dockerfiles. Use specific base image versions for reproducible builds."
)
def avoid_zypper_update():
    return instruction(type="RUN", contains="zypper update")
