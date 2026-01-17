"""
DOCKER-BP-025: Missing -y flag for yum

This rule detects yum install commands without the -y flag, which can cause
builds to fail in automated environments where user interaction is not possible.

VULNERABLE EXAMPLE:
```dockerfile
FROM centos:7

# Bad: Missing -y flag
# Build will hang waiting for confirmation
RUN yum install httpd mod_ssl
```

SECURE EXAMPLE:
```dockerfile
FROM centos:7

# Good: Using -y flag for non-interactive installation
RUN yum install -y httpd mod_ssl && \
    yum clean all && \
    rm -rf /var/cache/yum
```
"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction
from rules.container_combinators import all_of


@dockerfile_rule(
    id="DOCKER-BP-025",
    name="Missing -y flag for yum",
    severity="LOW",
    cwe="CWE-710",
    category="best-practice",
    tags="docker,dockerfile,yum,package-manager,centos,rhel,automation,ci-cd,build,best-practice,non-interactive",
    message="yum install without -y flag. Add -y for non-interactive builds."
)
def missing_yum_assume_yes():
    return all_of(
        instruction(type="RUN", contains="yum install"),
        instruction(type="RUN", not_contains="-y")
    )
