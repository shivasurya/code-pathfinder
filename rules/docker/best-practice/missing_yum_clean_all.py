"""
DOCKER-BP-012: Missing yum clean all

Security Impact: LOW
Best Practice Violation

DESCRIPTION:
Detects 'yum install' commands without subsequent 'yum clean all'.
Package manager caches unnecessarily increase image size by storing package metadata
and repository information that is not needed at runtime.

WHY THIS IS PROBLEMATIC:
1. Increased Image Size: Package caches can add hundreds of MBs
2. Wasted Storage: Cache files are not used after build
3. Slower Deployments: Larger images take longer to transfer
4. Higher Costs: More storage and bandwidth usage

VULNERABLE EXAMPLE:
```dockerfile
FROM centos:8

# Bad: Leaves yum cache, increases image size
RUN yum install -y nginx
```

SECURE EXAMPLE:
```dockerfile
FROM centos:8

# Good: Cleans cache in same layer
RUN yum install -y nginx && yum clean all
```

REMEDIATION:
Always run 'yum clean all' after 'yum install' in the same RUN instruction to
remove package cache and reduce final image size.

REFERENCES:
- Docker Best Practices
- hadolint DL3038
"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction
from rules.container_combinators import all_of


@dockerfile_rule(
    id="DOCKER-BP-012",
    name="Missing yum clean all",
    severity="LOW",
    cwe="CWE-710",
    category="best-practice",
    tags="docker,dockerfile,yum,package-manager,centos,rhel,cache,cleanup,image-size,optimization,best-practice",
    message="RUN instruction uses 'yum install' without 'yum clean all'. This leaves package cache and increases image size."
)
def missing_yum_clean_all():
    return all_of(
        instruction(type="RUN", contains="yum install"),
        instruction(type="RUN", not_contains="yum clean all")
    )
