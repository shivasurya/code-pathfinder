"""
DOCKER-BP-030: Nonsensical Command (cd in same RUN)

Security Impact: LOW
Best Practice Violation

DESCRIPTION:
Detects 'cd' command appearing in the middle or end of a RUN instruction
chain (after ; or &&). Using cd this way is confusing and indicates the
developer may not understand that WORKDIR should be used instead.

WHY THIS IS PROBLEMATIC:
1. Confusing Intent: Unclear what the cd is meant to accomplish
2. Potential Bug: May indicate misunderstanding of how cd works
3. Better Alternative: WORKDIR is the proper way to change directories
4. Less Maintainable: Makes Dockerfile harder to understand
5. Error Prone: Easy to make mistakes with cd in chains

VULNERABLE EXAMPLE:
```dockerfile
FROM ubuntu:22.04

# Bad: cd in chain is confusing and pointless
RUN apt-get update && cd /tmp && apt-get install -y nginx
RUN mkdir /app; cd /app; touch file.txt
```

SECURE EXAMPLE:
```dockerfile
FROM ubuntu:22.04

# Good: Use WORKDIR to change directories
RUN apt-get update && apt-get install -y nginx
WORKDIR /app
RUN touch file.txt
```

REMEDIATION:
Replace chained cd commands with WORKDIR instructions. WORKDIR is explicit,
clear, and persists across instructions.

If you need to run a command in a different directory temporarily:
```dockerfile
# OK: cd at start of command for temporary directory change
RUN cd /tmp && ./configure && make && make install
# Or better: use subshell
RUN (cd /tmp && ./configure && make && make install)
```

REFERENCES:
- Docker Best Practices
- hadolint DL4006
"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction
from rules.container_combinators import any_of


@dockerfile_rule(
    id="DOCKER-BP-030",
    name="Nonsensical Command",
    severity="LOW",
    cwe="CWE-710",
    category="best-practice",
    tags="docker,dockerfile,cd,workdir,directory,shell,best-practice,anti-pattern,confusing",
    message="RUN command uses 'cd' which doesn't persist. Use WORKDIR instead."
)
def nonsensical_command():
    return any_of(
        instruction(type="RUN", regex=r";\s*cd\s+"),
        instruction(type="RUN", regex=r"&&\s*cd\s+")
    )
