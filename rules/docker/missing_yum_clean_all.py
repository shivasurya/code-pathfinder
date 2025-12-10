"""
DOCKER-BP-012: Missing yum clean all

Best Practice: Clean package manager cache

DESCRIPTION:
Detects yum install commands without subsequent 'yum clean all'.
Package manager caches unnecessarily increase image size.

EXAMPLE:
```dockerfile
# Bad - leaves cache
RUN yum install -y nginx

# Good - cleans cache
RUN yum install -y nginx && yum clean all
```

REMEDIATION:
Always run 'yum clean all' after 'yum install' in the same RUN instruction.
"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction
from rules.container_combinators import all_of


@dockerfile_rule(
    id="DOCKER-BP-012",
    name="Missing yum clean all",
    severity="LOW",
    category="best-practice",
    message="RUN instruction uses 'yum install' without 'yum clean all'. This leaves package cache and increases image size."
)
def missing_yum_clean_all():
    return all_of(
        instruction(type="RUN", contains="yum install"),
        instruction(type="RUN", not_contains="yum clean all")
    )
