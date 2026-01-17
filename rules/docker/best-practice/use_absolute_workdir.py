"""
DOCKER-BP-018: Use Absolute Path in WORKDIR

Security Impact: LOW
Best Practice Violation

DESCRIPTION:
Detects WORKDIR instructions using relative paths instead of absolute paths.
Relative paths can lead to confusion about the actual working directory and
make Dockerfiles harder to understand and maintain.

WHY THIS IS PROBLEMATIC:
1. Ambiguous Location: Unclear where directory is relative to
2. Error Prone: Easy to lose track of current directory
3. Less Readable: Makes Dockerfile harder to understand
4. Debugging Difficulty: Hard to troubleshoot path issues
5. Stacking Behavior: Multiple relative WORKDIRs stack unpredictably

VULNERABLE EXAMPLE:
```dockerfile
FROM node:18

# Bad: Relative path - where is this relative to?
WORKDIR app
WORKDIR src  # Now at some-unknown-path/app/src
```

SECURE EXAMPLE:
```dockerfile
FROM node:18

# Good: Absolute path - clear and unambiguous
WORKDIR /app
WORKDIR /app/src  # Clear full path
```

REMEDIATION:
Always use absolute paths (starting with /) in WORKDIR instructions.
This makes the working directory explicit and prevents confusion.

REFERENCES:
- Docker Best Practices
- hadolint DL3000
"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction


@dockerfile_rule(
    id="DOCKER-BP-018",
    name="Use Absolute Path in WORKDIR",
    severity="LOW",
    cwe="CWE-710",
    category="best-practice",
    tags="docker,dockerfile,workdir,path,absolute-path,best-practice,clarity,maintainability,filesystem",
    message="WORKDIR should use absolute paths starting with /."
)
def use_absolute_workdir():
    return instruction(type="WORKDIR", workdir_not_absolute=True)
