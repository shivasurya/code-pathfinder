"""
DOCKER-BP-015: Missing Image Version

Best Practice: Always specify image versions

DESCRIPTION:
Detects FROM instructions using 'latest' tag or no tag at all.
Using latest or untagged images creates non-reproducible builds and
potential security/stability issues.

VULNERABLE EXAMPLE:
```dockerfile
FROM ubuntu          # No tag = latest
FROM nginx:latest    # Explicit latest
```

SECURE EXAMPLE:
```dockerfile
FROM ubuntu:22.04
FROM nginx:1.24.0-alpine
```

REMEDIATION:
Always specify explicit version tags for base images.
"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction
from rules.container_combinators import any_of


@dockerfile_rule(
    id="DOCKER-BP-015",
    name="Missing Image Version",
    severity="HIGH",
    cwe="CWE-1188",
    category="best-practice",
    tags="docker,dockerfile,from,image,tag,version,latest,reproducibility,best-practice,supply-chain,dependency-management",
    message="FROM instruction uses 'latest' tag or no tag. Specify explicit versions for reproducible builds."
)
def missing_image_version():
    return any_of(
        instruction(type="FROM", image_tag="latest"),
        instruction(type="FROM", missing_digest=True)
    )
