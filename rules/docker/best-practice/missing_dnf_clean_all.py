"""
DOCKER-BP-013: Missing dnf clean all

Security Impact: LOW
Best Practice Violation

DESCRIPTION:
Detects 'dnf install' commands without subsequent 'dnf clean all'.
Package manager caches unnecessarily increase image size by storing package metadata
and repository information that is not needed at runtime.

WHY THIS IS PROBLEMATIC:
1. Increased Image Size: Package caches can add hundreds of MBs
2. Wasted Storage: Cache files are not used after build
3. Slower Deployments: Larger images take longer to transfer
4. Higher Costs: More storage and bandwidth usage

VULNERABLE EXAMPLE:
```dockerfile
FROM fedora:38

# Bad: Leaves dnf cache, increases image size
RUN dnf install -y nginx
```

SECURE EXAMPLE:
```dockerfile
FROM fedora:38

# Good: Cleans cache in same layer
RUN dnf install -y nginx && dnf clean all
```

REMEDIATION:
Always run 'dnf clean all' after 'dnf install' in the same RUN instruction to
remove package cache and reduce final image size.

REFERENCES:
- Docker Best Practices
- hadolint DL3038
"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction
from rules.container_combinators import all_of


@dockerfile_rule(
    id="DOCKER-BP-013",
    name="Missing dnf clean all",
    severity="LOW",
    cwe="CWE-710",
    category="best-practice",
    tags="docker,dockerfile,dnf,package-manager,fedora,rhel,cache,cleanup,image-size,optimization,best-practice",
    message="RUN uses 'dnf install' without 'dnf clean all'. This increases image size."
)
def missing_dnf_clean_all():
    return all_of(
        instruction(type="RUN", contains="dnf install"),
        instruction(type="RUN", not_contains="dnf clean all")
    )
