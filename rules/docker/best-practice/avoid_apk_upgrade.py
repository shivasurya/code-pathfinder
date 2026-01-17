"""
DOCKER-BP-028: Avoid apk upgrade in Dockerfile

Using 'apk upgrade' in Dockerfiles breaks image reproducibility and can introduce
unexpected changes. Use specific base image versions instead.

VULNERABLE EXAMPLE:
```dockerfile
FROM alpine:3.19

# Bad: Using apk upgrade
# Makes builds non-reproducible
RUN apk update && apk upgrade
RUN apk add nginx
```

SECURE EXAMPLE:
```dockerfile
# Good: Use specific base image version
FROM alpine:3.19.0

# Install packages without upgrading
RUN apk add --no-cache nginx=1.24.0-r15

# If you need latest packages, update the base image version
# FROM alpine:3.20.0
```
"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction


@dockerfile_rule(
    id="DOCKER-BP-028",
    name="Avoid apk upgrade",
    severity="MEDIUM",
    cwe="CWE-710",
    category="best-practice",
    tags="docker,dockerfile,apk,package-manager,alpine,upgrade,reproducibility,best-practice,anti-pattern",
    message="Avoid 'apk upgrade' in Dockerfiles. Use specific base image versions instead for reproducible builds."
)
def avoid_apk_upgrade():
    return instruction(type="RUN", contains="apk upgrade")
