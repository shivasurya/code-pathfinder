"""
DOCKER-BP-021: Missing -y flag for apt-get

This rule detects apt-get install commands without the -y flag, which can cause
builds to fail in automated environments where user interaction is not possible.

VULNERABLE EXAMPLE:
```dockerfile
FROM ubuntu:22.04

# Bad: Missing -y flag
# Build will hang waiting for user confirmation
RUN apt-get update
RUN apt-get install nginx curl

# This will fail in CI/CD pipelines
```

SECURE EXAMPLE:
```dockerfile
FROM ubuntu:22.04

# Good: Using -y flag for non-interactive installation
RUN apt-get update && \
    apt-get install -y nginx curl && \
    rm -rf /var/lib/apt/lists/*

# Alternative: Using --yes flag
RUN apt-get update && \
    apt-get install --yes \
        nginx \
        curl \
        ca-certificates && \
    rm -rf /var/lib/apt/lists/*
```
"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction
from rules.container_combinators import all_of


@dockerfile_rule(
    id="DOCKER-BP-021",
    name="Missing -y flag for apt-get",
    severity="LOW",
    cwe="CWE-710",
    category="best-practice",
    tags="docker,dockerfile,apt-get,package-manager,automation,ci-cd,build,ubuntu,debian,best-practice,non-interactive",
    message="apt-get install without -y flag. Add -y or --yes for non-interactive builds."
)
def missing_apt_assume_yes():
    return all_of(
        instruction(type="RUN", contains="apt-get install"),
        instruction(type="RUN", not_contains="-y"),
        instruction(type="RUN", not_contains="--yes")
    )
