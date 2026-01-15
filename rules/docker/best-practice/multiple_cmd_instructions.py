"""
DOCKER-COR-003: Multiple CMD Instructions

Correctness: Only last CMD takes effect

DESCRIPTION:
Detects multiple CMD instructions. Only the last one is used.

EXAMPLE:
```dockerfile
CMD ["echo", "first"]   # Ignored
CMD ["echo", "second"]  # Used
```
"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction


@dockerfile_rule(
    id="DOCKER-COR-003",
    name="Multiple CMD Instructions",
    severity="MEDIUM",
    category="correctness",
    message="Multiple CMD instructions detected. Only the last one takes effect."
)
def multiple_cmd_instructions():
    return instruction(type="CMD")
