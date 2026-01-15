"""
DOCKER-BP-016: Prefer JSON Notation for CMD/ENTRYPOINT

Best Practice: Use exec form over shell form

DESCRIPTION:
Shell form wraps commands in /bin/sh -c, which:
- Doesn't pass signals correctly
- Creates an extra shell process
- Makes PID 1 the shell instead of your app

EXAMPLE:
```dockerfile
# Shell form - not recommended
CMD nginx -g "daemon off;"

# Exec form (JSON) - recommended
CMD ["nginx", "-g", "daemon off;"]
```
"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction
from rules.container_combinators import any_of


@dockerfile_rule(
    id="DOCKER-BP-016",
    name="Prefer JSON Notation for CMD/ENTRYPOINT",
    severity="LOW",
    category="best-practice",
    message="Use JSON notation (exec form) for CMD/ENTRYPOINT for proper signal handling."
)
def prefer_json_notation():
    return any_of(
        instruction(type="CMD", command_form="shell"),
        instruction(type="ENTRYPOINT", command_form="shell")
    )
