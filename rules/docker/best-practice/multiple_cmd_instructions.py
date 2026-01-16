"""
DOCKER-COR-003: Multiple CMD Instructions

Security Impact: MEDIUM
Correctness Issue

DESCRIPTION:
Detects Dockerfiles with multiple CMD instructions. Docker only honors
the last CMD, making earlier ones misleading and potentially causing
unexpected runtime behavior. This can lead to confusion and bugs.

WHY THIS IS PROBLEMATIC:
1. Only Last Used: Docker ignores all but the last CMD
2. Misleading Code: Earlier CMDs appear to work but don't
3. Debugging Difficulty: Unclear which CMD is actually used
4. Potential Bugs: May run wrong command at runtime
5. Maintenance Issues: Confusing for developers reading the Dockerfile

VULNERABLE EXAMPLE:
```dockerfile
FROM ubuntu:22.04

# Bad: Multiple CMDs - only last one is used
CMD ["echo", "first"]       # Ignored silently
CMD ["python3", "app.py"]   # Ignored silently
CMD ["nginx", "-g", "daemon off;"]  # Only this one takes effect
```

SECURE EXAMPLE:
```dockerfile
FROM ubuntu:22.04

# Good: Single CMD per Dockerfile
CMD ["nginx", "-g", "daemon off;"]

# Or use multi-stage builds if you need different commands:
FROM ubuntu:22.04 AS builder
CMD ["make", "build"]

FROM ubuntu:22.04 AS runtime
CMD ["nginx", "-g", "daemon off;"]
```

REMEDIATION:
Remove all but one CMD instruction. If you need different commands for
different scenarios, use:
1. Multi-stage builds with different CMD per stage
2. ENTRYPOINT + CMD: ENTRYPOINT defines executable, CMD defines default args
3. Runtime override: docker run image [custom-command]

REFERENCES:
- Docker Documentation: CMD instruction
- hadolint DL4004
- Docker Best Practices
"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction


@dockerfile_rule(
    id="DOCKER-COR-003",
    name="Multiple CMD Instructions",
    severity="MEDIUM",
    cwe="CWE-710",
    category="correctness",
    tags="docker,dockerfile,cmd,correctness,configuration,maintainability,confusing,anti-pattern",
    message="Multiple CMD instructions detected. Only the last one takes effect."
)
def multiple_cmd_instructions():
    return instruction(type="CMD")
