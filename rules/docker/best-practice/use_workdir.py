"""
DOCKER-BP-017: Use WORKDIR Instead of cd

Security Impact: LOW
Best Practice Violation

DESCRIPTION:
Detects use of 'cd' command in RUN instructions when WORKDIR is not used.
Using 'cd' in RUN commands is error-prone, less clear, and doesn't persist
across instructions. WORKDIR is the proper way to set working directory.

WHY THIS IS PROBLEMATIC:
1. Doesn't Persist: cd only affects current RUN command
2. Error Prone: Easy to forget cd in subsequent commands
3. Less Readable: Makes Dockerfile harder to understand
4. Fragile: Breaks if directory structure changes
5. Chaining Required: Must use && to chain commands

VULNERABLE EXAMPLE:
```dockerfile
FROM node:18

COPY . .

# Bad: cd doesn't persist, must chain all commands
RUN cd /app && npm install
RUN cd /app && npm build  # Must repeat cd
```

SECURE EXAMPLE:
```dockerfile
FROM node:18

# Good: WORKDIR persists across all subsequent instructions
WORKDIR /app
COPY . .
RUN npm install
RUN npm build  # Already in /app directory
```

REMEDIATION:
Replace 'cd' commands with WORKDIR instruction. WORKDIR creates the directory
if it doesn't exist and sets it as the working directory for all subsequent
instructions (RUN, CMD, ENTRYPOINT, COPY, ADD).

REFERENCES:
- Docker Best Practices
- hadolint DL3003
"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction, missing
from rules.container_combinators import all_of, any_of


@dockerfile_rule(
    id="DOCKER-BP-017",
    name="Use WORKDIR Instead of cd",
    severity="LOW",
    cwe="CWE-710",
    category="best-practice",
    tags="docker,dockerfile,workdir,cd,directory,best-practice,maintainability,clarity,anti-pattern",
    message="Use WORKDIR instruction instead of 'cd' in RUN commands."
)
def use_workdir():
    return all_of(
        any_of(
            instruction(type="RUN", contains=" cd "),
            instruction(type="RUN", regex=r"\bcd\s+")
        ),
        missing(instruction="WORKDIR")
    )
