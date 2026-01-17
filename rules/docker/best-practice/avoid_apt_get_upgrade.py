"""
DOCKER-BP-006: Avoid apt-get upgrade in Dockerfile

Security Impact: MEDIUM
Best Practice Violation

DESCRIPTION:
Detects use of `apt-get upgrade` or `apt-get dist-upgrade` in Dockerfiles.
Running system upgrades in Docker builds creates unpredictable, non-reproducible images
and can introduce breaking changes or security vulnerabilities.

WHY THIS IS PROBLEMATIC:
1. **Non-Reproducible Builds**: Different build times produce different images
2. **Breaks Caching**: Forces rebuild of all subsequent layers
3. **Unpredictable Versions**: Can install untested package versions
4. **Security Risk**: May introduce vulnerabilities or break dependencies

SECURE ALTERNATIVE:
Use specific base image tags with known package versions, then install only needed packages.

VULNERABLE EXAMPLE:
```dockerfile
FROM ubuntu:latest
RUN apt-get update && apt-get upgrade -y  # ❌ Bad practice
```

SECURE EXAMPLE:
```dockerfile
FROM ubuntu:22.04  # Specific version
RUN apt-get update && apt-get install -y nginx=1.18.0-0ubuntu1
```

REMEDIATION:
1. Remove apt-get upgrade commands
2. Use specific base image versions
3. Install specific package versions when needed
4. Rebuild images periodically with updated base images

REFERENCES:
- Docker Best Practices
- hadolint DL3005
"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction
from rules.container_combinators import any_of


@dockerfile_rule(
    id="DOCKER-BP-006",
    name="Avoid apt-get upgrade",
    severity="MEDIUM",
    cwe="CWE-710",
    category="best-practice",
    tags="docker,dockerfile,apt-get,upgrade,package-manager,ubuntu,debian,reproducibility,best-practice,anti-pattern,build",
    message="Avoid apt-get upgrade in Dockerfiles. Use specific base image versions instead."
)
def avoid_apt_get_upgrade():
    return any_of(
        instruction(type="RUN", contains="apt-get upgrade"),
        instruction(type="RUN", contains="apt-get dist-upgrade")
    )
