"""
DOCKER-SEC-008: Missing USER Before ENTRYPOINT

Security: USER instruction before ENTRYPOINT

DESCRIPTION:
Detects ENTRYPOINT without a preceding USER instruction.
Ensures the entrypoint doesn't run as root.

EXAMPLE:
```dockerfile
# Bad - runs as root
ENTRYPOINT ["/app/start.sh"]

# Good - runs as non-root
USER appuser
ENTRYPOINT ["/app/start.sh"]
```
"""

from rules.container_decorators import dockerfile_rule
from rules.container_combinators import all_of
from rules.container_matchers import instruction, missing


@dockerfile_rule(
    id="DOCKER-SEC-008",
    name="Missing USER Before ENTRYPOINT",
    severity="MEDIUM",
    category="security",
    message="ENTRYPOINT without preceding USER instruction. Container may run as root."
)
def missing_user_entrypoint():
    return all_of(
        instruction(type="ENTRYPOINT"),
        missing(instruction="USER")
    )
