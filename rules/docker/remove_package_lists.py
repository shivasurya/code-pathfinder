"""
DOCKER-BP-014: Remove apt Package Lists

Best Practice: Clean apt package lists after installation

DESCRIPTION:
Detects apt-get install without removing /var/lib/apt/lists/*.
Package lists are not needed at runtime and waste space.

SECURE EXAMPLE:
```dockerfile
RUN apt-get update && apt-get install -y nginx \\
    && rm -rf /var/lib/apt/lists/*
```

REMEDIATION:
Add 'rm -rf /var/lib/apt/lists/*' after apt-get install.
"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction
from rules.container_combinators import all_of


@dockerfile_rule(
    id="DOCKER-BP-014",
    name="Remove apt Package Lists",
    severity="LOW",
    category="best-practice",
    message="apt-get install without removing /var/lib/apt/lists/*. This wastes image space."
)
def remove_package_lists():
    return all_of(
        instruction(type="RUN", contains="apt-get install"),
        instruction(type="RUN", not_contains="/var/lib/apt/lists/")
    )
