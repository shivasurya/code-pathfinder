"""
DOCKER-BP-014: Remove apt Package Lists

Security Impact: LOW
Best Practice Violation

DESCRIPTION:
Detects 'apt-get install' without removing /var/lib/apt/lists/*.
Package lists contain repository metadata that is not needed at runtime and wastes space.
This is specific to Debian/Ubuntu based images.

WHY THIS IS PROBLEMATIC:
1. Increased Image Size: Package lists can add tens to hundreds of MBs
2. Wasted Storage: Metadata files are not used after build
3. Slower Deployments: Larger images take longer to transfer
4. Higher Costs: More storage and bandwidth usage

VULNERABLE EXAMPLE:
```dockerfile
FROM ubuntu:22.04

# Bad: Leaves package lists, increases image size
RUN apt-get update && apt-get install -y nginx
```

SECURE EXAMPLE:
```dockerfile
FROM ubuntu:22.04

# Good: Removes package lists in same layer
RUN apt-get update && apt-get install -y nginx \\
    && rm -rf /var/lib/apt/lists/*
```

REMEDIATION:
Add 'rm -rf /var/lib/apt/lists/*' after apt-get install in the same RUN instruction
to remove package metadata and reduce final image size.

REFERENCES:
- Docker Best Practices
- hadolint DL3009
"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction
from rules.container_combinators import all_of


@dockerfile_rule(
    id="DOCKER-BP-014",
    name="Remove apt Package Lists",
    severity="LOW",
    cwe="CWE-710",
    category="best-practice",
    tags="docker,dockerfile,apt-get,package-manager,ubuntu,debian,cache,cleanup,image-size,optimization,best-practice",
    message="apt-get install without removing /var/lib/apt/lists/*. This wastes image space."
)
def remove_package_lists():
    return all_of(
        instruction(type="RUN", contains="apt-get install"),
        instruction(type="RUN", not_contains="/var/lib/apt/lists/")
    )
