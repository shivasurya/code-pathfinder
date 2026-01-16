"""
DOCKER-BP-016: Prefer JSON Notation for CMD/ENTRYPOINT

Security Impact: LOW
Best Practice Violation

DESCRIPTION:
Detects CMD or ENTRYPOINT using shell form instead of exec form (JSON array).
Shell form wraps commands in /bin/sh -c, which creates issues with signal handling,
process management, and adds an unnecessary shell layer.

WHY THIS IS PROBLEMATIC:
1. Signal Handling: Shell doesn't forward signals (SIGTERM, SIGINT) correctly
2. PID 1 Issues: Shell becomes PID 1 instead of your application
3. Zombie Processes: Shell may not reap child processes properly
4. Extra Process: Unnecessary shell layer wastes resources
5. Graceful Shutdown: Container may not stop cleanly

VULNERABLE EXAMPLE:
```dockerfile
FROM nginx:alpine

# Bad: Shell form - signals not handled correctly
CMD nginx -g "daemon off;"
ENTRYPOINT /app/start.sh
```

SECURE EXAMPLE:
```dockerfile
FROM nginx:alpine

# Good: Exec form (JSON) - proper signal handling
CMD ["nginx", "-g", "daemon off;"]
ENTRYPOINT ["/app/start.sh"]
```

REMEDIATION:
Convert CMD and ENTRYPOINT instructions to exec form using JSON array syntax:
- Shell form: CMD command arg1 arg2
- Exec form: CMD ["command", "arg1", "arg2"]

REFERENCES:
- Docker Best Practices
- hadolint DL3025
- Docker ENTRYPOINT documentation
"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction
from rules.container_combinators import any_of


@dockerfile_rule(
    id="DOCKER-BP-016",
    name="Prefer JSON Notation for CMD/ENTRYPOINT",
    severity="LOW",
    cwe="CWE-710",
    category="best-practice",
    tags="docker,dockerfile,cmd,entrypoint,exec-form,json,signal-handling,best-practice,process-management,pid1",
    message="Use JSON notation (exec form) for CMD/ENTRYPOINT for proper signal handling."
)
def prefer_json_notation():
    return any_of(
        instruction(type="CMD", command_form="shell"),
        instruction(type="ENTRYPOINT", command_form="shell")
    )
