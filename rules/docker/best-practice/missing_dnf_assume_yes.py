"""
DOCKER-BP-026: Missing -y flag for dnf

This rule detects dnf install commands without the -y flag, which can cause
builds to fail in automated environments where user interaction is not possible.

VULNERABLE EXAMPLE:
```dockerfile
FROM fedora:39

# Bad: Missing -y flag
# Build will hang waiting for confirmation
RUN dnf install nginx
```

SECURE EXAMPLE:
```dockerfile
FROM fedora:39

# Good: Using -y flag for non-interactive installation
RUN dnf install -y nginx && \
    dnf clean all && \
    rm -rf /var/cache/dnf
```
"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction
from rules.container_combinators import all_of


@dockerfile_rule(
    id="DOCKER-BP-026",
    name="Missing -y flag for dnf",
    severity="LOW",
    cwe="CWE-710",
    category="best-practice",
    tags="docker,dockerfile,dnf,package-manager,fedora,rhel,automation,ci-cd,build,best-practice,non-interactive",
    message="dnf install without -y flag. Add -y for non-interactive builds."
)
def missing_dnf_assume_yes():
    return all_of(
        instruction(type="RUN", contains="dnf install"),
        instruction(type="RUN", not_contains="-y")
    )
