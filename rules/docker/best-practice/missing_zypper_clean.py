"""
DOCKER-BP-020: Missing zypper clean

Security Impact: LOW
Best Practice Violation

DESCRIPTION:
Detects 'zypper install' commands without subsequent 'zypper clean'.
Package manager caches unnecessarily increase image size by storing package metadata
and repository information that is not needed at runtime.

WHY THIS IS PROBLEMATIC:
1. Increased Image Size: Package caches can add hundreds of MBs
2. Wasted Storage: Cache files are not used after build
3. Slower Deployments: Larger images take longer to transfer
4. Higher Costs: More storage and bandwidth usage

VULNERABLE EXAMPLE:
```dockerfile
FROM opensuse/leap:15.4

# Bad: Leaves zypper cache, increases image size
RUN zypper install -y nginx
```

SECURE EXAMPLE:
```dockerfile
FROM opensuse/leap:15.4

# Good: Cleans cache in same layer
RUN zypper install -y nginx && zypper clean
```

REMEDIATION:
Always run 'zypper clean' after 'zypper install' in the same RUN instruction to
remove package cache and reduce final image size.

REFERENCES:
- Docker Best Practices
- hadolint DL3036
"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction
from rules.container_combinators import all_of


@dockerfile_rule(
    id="DOCKER-BP-020",
    name="Missing zypper clean",
    severity="LOW",
    cwe="CWE-710",
    category="best-practice",
    tags="docker,dockerfile,zypper,package-manager,opensuse,suse,cache,cleanup,image-size,optimization,best-practice",
    message="RUN uses 'zypper install' without 'zypper clean'. This increases image size."
)
def missing_zypper_clean():
    return all_of(
        instruction(type="RUN", contains="zypper install"),
        instruction(type="RUN", not_contains="zypper clean")
    )
