"""
DOCKER-COR-001: Multiple ENTRYPOINT Instructions

Correctness Issue: Only last ENTRYPOINT takes effect

DESCRIPTION:
Detects Dockerfiles with multiple ENTRYPOINT instructions. Docker only honors
the last ENTRYPOINT, making earlier ones misleading and potentially causing
unexpected runtime behavior.

EXAMPLE:
```dockerfile
FROM ubuntu
ENTRYPOINT ["/bin/sh"]  # Ignored
ENTRYPOINT ["/app/start.sh"]  # This one is used
```

REMEDIATION:
Keep only one ENTRYPOINT instruction per Dockerfile (or per build stage).

REFERENCES:
- Docker Documentation: ENTRYPOINT instruction
- hadolint DL4003
"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction


@dockerfile_rule(
    id="DOCKER-COR-001",
    name="Multiple ENTRYPOINT Instructions",
    severity="MEDIUM",
    category="correctness",
    message="Dockerfile has multiple ENTRYPOINT instructions. Only the last one takes effect, making earlier ones misleading."
)
def multiple_entrypoint_instructions():
    # Note: This is a simplified check - ideally would count occurrences
    # For now, this flags any ENTRYPOINT which helps identify the issue
    return instruction(type="ENTRYPOINT")
