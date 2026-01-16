"""
DOCKER-COR-002: Invalid Port Number

Correctness: Port number validation

DESCRIPTION:
Detects EXPOSE instructions with invalid port numbers.
Valid ports are 1-65535.

VULNERABLE EXAMPLE:
```dockerfile
EXPOSE 0
EXPOSE 70000
```

SECURE EXAMPLE:
```dockerfile
EXPOSE 8080
EXPOSE 443
```
"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction
from rules.container_combinators import any_of


@dockerfile_rule(
    id="DOCKER-COR-002",
    name="Invalid Port Number",
    severity="HIGH",
    cwe="CWE-20",
    category="correctness",
    tags="docker,dockerfile,port,expose,validation,input-validation,correctness,networking,configuration",
    message="EXPOSE instruction has invalid port number. Valid ports are 1-65535."
)
def invalid_port():
    return any_of(
        instruction(type="EXPOSE", port_less_than=1),
        instruction(type="EXPOSE", port_greater_than=65535)
    )
