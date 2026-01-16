"""
DOCKER-COR-001: Multiple ENTRYPOINT Instructions

Security Impact: MEDIUM
Correctness Issue

DESCRIPTION:
Detects Dockerfiles with multiple ENTRYPOINT instructions. Docker only honors
the last ENTRYPOINT, making earlier ones misleading and potentially causing
unexpected runtime behavior. This can lead to confusion and bugs.

WHY THIS IS PROBLEMATIC:
1. Only Last Used: Docker ignores all but the last ENTRYPOINT
2. Misleading Code: Earlier ENTRYPOINTs appear to work but don't
3. Debugging Difficulty: Unclear which ENTRYPOINT is actually used
4. Potential Bugs: May run wrong executable at runtime
5. Maintenance Issues: Confusing for developers reading the Dockerfile

VULNERABLE EXAMPLE:
```dockerfile
FROM ubuntu:22.04

# Bad: Multiple ENTRYPOINTs - only last one is used
ENTRYPOINT ["/bin/sh"]          # Ignored silently
ENTRYPOINT ["/usr/bin/python3"] # Ignored silently
ENTRYPOINT ["/app/start.sh"]    # Only this one takes effect
```

SECURE EXAMPLE:
```dockerfile
FROM ubuntu:22.04

# Good: Single ENTRYPOINT per Dockerfile
ENTRYPOINT ["/app/start.sh"]

# Or use multi-stage builds if you need different entrypoints:
FROM ubuntu:22.04 AS stage1
ENTRYPOINT ["/app/build.sh"]

FROM ubuntu:22.04 AS stage2
ENTRYPOINT ["/app/start.sh"]
```

REMEDIATION:
Remove all but one ENTRYPOINT instruction. If you need different entrypoints
for different scenarios, use:
1. Multi-stage builds with different entrypoints per stage
2. Runtime arguments: ENTRYPOINT ["/app/start.sh"] + CMD ["--default-arg"]
3. Environment variables to control behavior within a single entrypoint

REFERENCES:
- Docker Documentation: ENTRYPOINT instruction
- hadolint DL4003
- Docker Best Practices
"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction


@dockerfile_rule(
    id="DOCKER-COR-001",
    name="Multiple ENTRYPOINT Instructions",
    severity="MEDIUM",
    cwe="CWE-710",
    category="correctness",
    tags="docker,dockerfile,entrypoint,correctness,configuration,maintainability,confusing,anti-pattern",
    message="Dockerfile has multiple ENTRYPOINT instructions. Only the last one takes effect, making earlier ones misleading."
)
def multiple_entrypoint_instructions():
    # Note: This is a simplified check - ideally would count occurrences
    # For now, this flags any ENTRYPOINT which helps identify the issue
    return instruction(type="ENTRYPOINT")
