"""
DOCKER-BP-013: Missing dnf clean all

Best Practice: Clean package manager cache

DESCRIPTION:
Detects dnf install commands without subsequent 'dnf clean all'.
Package manager caches unnecessarily increase image size.

REMEDIATION:
Always run 'dnf clean all' after 'dnf install' in the same RUN instruction.

EXAMPLE:
```dockerfile
RUN dnf install -y nginx && dnf clean all
```
"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction
from rules.container_combinators import all_of


@dockerfile_rule(
    id="DOCKER-BP-013",
    name="Missing dnf clean all",
    severity="LOW",
    category="best-practice",
    message="RUN uses 'dnf install' without 'dnf clean all'. This increases image size."
)
def missing_dnf_clean_all():
    return all_of(
        instruction(type="RUN", contains="dnf install"),
        instruction(type="RUN", not_contains="dnf clean all")
    )
