"""
DOCKER-BP-018: Use Absolute Path in WORKDIR

Best Practice: WORKDIR should use absolute paths

DESCRIPTION:
Relative paths in WORKDIR can be confusing and error-prone.

EXAMPLE:
```dockerfile
# Bad - relative path
WORKDIR app

# Good - absolute path
WORKDIR /app
```
"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction


@dockerfile_rule(
    id="DOCKER-BP-018",
    name="Use Absolute Path in WORKDIR",
    severity="LOW",
    category="best-practice",
    message="WORKDIR should use absolute paths starting with /."
)
def use_absolute_workdir():
    return instruction(type="WORKDIR", workdir_not_absolute=True)
