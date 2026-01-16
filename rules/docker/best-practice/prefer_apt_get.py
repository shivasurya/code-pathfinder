"""
DOCKER-BP-023: Prefer apt-get over apt

Security Impact: LOW
Best Practice Violation

DESCRIPTION:
Detects use of 'apt' command instead of 'apt-get' in Dockerfiles.
The 'apt' command is designed for interactive use and has an unstable CLI interface
that may change between versions, making builds less reproducible.

WHY THIS IS PROBLEMATIC:
1. Unstable Interface: apt output and behavior can change between versions
2. Non-Reproducible Builds: Different apt versions may behave differently
3. Script Instability: apt is not designed for scripting
4. Better Alternatives: apt-get has stable, well-documented interface

VULNERABLE EXAMPLE:
```dockerfile
FROM ubuntu:22.04

# Bad: apt is for interactive use, unstable in scripts
RUN apt update && apt install -y nginx
```

SECURE EXAMPLE:
```dockerfile
FROM ubuntu:22.04

# Good: apt-get has stable CLI for scripting
RUN apt-get update && apt-get install -y nginx
```

REMEDIATION:
Replace 'apt' with 'apt-get' in all Dockerfile RUN commands for better stability
and reproducibility.

REFERENCES:
- Docker Best Practices
- hadolint DL3027
- Debian apt vs apt-get documentation
"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction
from rules.container_combinators import all_of


@dockerfile_rule(
    id="DOCKER-BP-023",
    name="Prefer apt-get over apt",
    severity="LOW",
    cwe="CWE-710",
    category="best-practice",
    tags="docker,dockerfile,apt,apt-get,package-manager,ubuntu,debian,scripting,stability,reproducibility,best-practice",
    message="Use apt-get instead of apt for better script stability in Dockerfiles."
)
def prefer_apt_get():
    return all_of(
        instruction(type="RUN", regex=r"\bapt\s+install"),
        instruction(type="RUN", not_contains="apt-get")
    )
