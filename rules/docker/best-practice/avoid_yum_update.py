"""
DOCKER-BP-029: Avoid yum update in Dockerfile

Security Impact: MEDIUM
Best Practice Violation

DESCRIPTION:
Detects use of 'yum update' in Dockerfiles. Running system updates in Docker builds
creates unpredictable, non-reproducible images and can introduce breaking changes or
security vulnerabilities.

WHY THIS IS PROBLEMATIC:
1. Non-Reproducible Builds: Different build times produce different images
2. Breaks Caching: Forces rebuild of all subsequent layers
3. Unpredictable Versions: Can install untested package versions
4. Security Risk: May introduce vulnerabilities or break dependencies

VULNERABLE EXAMPLE:
```dockerfile
FROM centos:latest
RUN yum update -y  # Bad: Unpredictable, non-reproducible builds
RUN yum install -y httpd
```

SECURE EXAMPLE:
```dockerfile
FROM centos:8  # Specific version
RUN yum install -y httpd-2.4.37-43.module_el8.5.0
# Or install latest from specific base image
RUN yum install -y httpd
```

REMEDIATION:
1. Remove 'yum update' commands
2. Use specific base image versions (e.g., centos:8 instead of centos:latest)
3. Install specific package versions when needed
4. Rebuild images periodically with updated base images

REFERENCES:
- Docker Best Practices
- hadolint DL3005
"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction


@dockerfile_rule(
    id="DOCKER-BP-029",
    name="Avoid yum update",
    severity="MEDIUM",
    cwe="CWE-710",
    category="best-practice",
    tags="docker,dockerfile,yum,package-manager,centos,rhel,update,reproducibility,best-practice,anti-pattern",
    message="Avoid 'yum update' in Dockerfiles. Use specific base image versions for reproducible builds."
)
def avoid_yum_update():
    return instruction(type="RUN", contains="yum update")
