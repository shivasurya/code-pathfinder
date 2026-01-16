"""
DOCKER-BP-027: Avoid --platform Flag with FROM

Security Impact: LOW
Best Practice Violation

DESCRIPTION:
Detects use of --platform flag in FROM instructions. Hardcoding platform
reduces portability and prevents Docker from automatically selecting the
appropriate platform for multi-architecture builds.

WHY THIS IS PROBLEMATIC:
1. Reduced Portability: Image won't work on other architectures
2. Multi-Arch Issues: Breaks automatic platform selection
3. Build Failures: May fail on different host platforms
4. Maintenance Burden: Need separate Dockerfiles for each platform
5. Emulation Overhead: Forces unnecessary emulation on native platforms

VULNERABLE EXAMPLE:
```dockerfile
# Bad: Hardcoded platform prevents portability
FROM --platform=linux/amd64 ubuntu:22.04
RUN apt-get update && apt-get install -y nginx
```

SECURE EXAMPLE:
```dockerfile
# Good: Let Docker handle platform selection automatically
FROM ubuntu:22.04
RUN apt-get update && apt-get install -y nginx

# Docker automatically pulls correct image for:
# - linux/amd64 on x86_64 systems
# - linux/arm64 on ARM systems
```

REMEDIATION:
Remove --platform flag from FROM instructions and let Docker automatically
select the appropriate platform. For multi-architecture builds, use Docker
buildx with proper build targets instead of hardcoding platform.

If you absolutely need platform-specific behavior, use build arguments:
```dockerfile
ARG TARGETPLATFORM
FROM ubuntu:22.04
RUN echo "Building for $TARGETPLATFORM"
```

REFERENCES:
- Docker Best Practices
- Docker Buildx documentation
"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction


@dockerfile_rule(
    id="DOCKER-BP-027",
    name="Avoid --platform Flag with FROM",
    severity="LOW",
    cwe="CWE-710",
    category="best-practice",
    tags="docker,dockerfile,from,platform,multi-arch,portability,buildx,architecture,best-practice",
    message="FROM with --platform flag reduces portability. Let Docker handle platform selection."
)
def avoid_platform_with_from():
    return instruction(type="FROM", contains="--platform")
