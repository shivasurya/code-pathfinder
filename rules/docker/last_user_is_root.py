"""
DOCKER-SEC-001: Last USER Instruction is Root

Security Impact: HIGH
CWE: CWE-269 (Improper Privilege Management)

DESCRIPTION:
This rule detects when the final USER instruction in a Dockerfile sets the user to 'root'.
Running containers as root is a security hazard because if an attacker gains control,
they will have unrestricted root access within the container, which can often be leveraged
to escape to the host system.

SECURITY IMPLICATIONS:
Running as root increases the impact of:
- Container escape vulnerabilities
- Privilege escalation attacks
- File permission bypass
- Process capability abuse

Real-world attack scenario:
```bash
# Container running as root with vulnerability
# Attacker exploits CVE to get shell
# Already has root - no privilege escalation needed
# Can directly manipulate system files, install backdoors, etc.
```

VULNERABLE EXAMPLE:
```dockerfile
FROM ubuntu:latest

# Install dependencies as root
RUN apt-get update && apt-get install -y nginx

# Switch to root for final operations
USER root

# Application runs as root - VULNERABLE
CMD ["nginx", "-g", "daemon off;"]
```

SECURE EXAMPLE:
```dockerfile
FROM ubuntu:latest

# Install dependencies as root
RUN apt-get update && apt-get install -y nginx \\
    && useradd -r -s /bin/false nginx-user

# Switch to non-root user before CMD
USER nginx-user

# Application runs as non-root - SECURE
CMD ["nginx", "-g", "daemon off;"]
```

REMEDIATION:
1. Always set USER to a non-root user before CMD/ENTRYPOINT
2. Create dedicated application users with minimal privileges
3. Use numeric UIDs (e.g., USER 1000) for better portability
4. Never switch back to root after setting a non-root user

REFERENCES:
- CWE-269: Improper Privilege Management
- https://github.com/hadolint/hadolint/wiki/DL3002
- OWASP Docker Security Cheat Sheet
- CIS Docker Benchmark 4.1
"""

from rules.container_decorators import dockerfile_rule
from rules.container_combinators import final_stage_has
from rules.container_matchers import instruction


@dockerfile_rule(
    id="DOCKER-SEC-009",
    name="Last USER Instruction is Root",
    severity="HIGH",
    cwe="CWE-269",
    category="security",
    message="The last USER instruction in the Dockerfile is 'root'. Switch to a "
            "non-root user after running privileged commands to reduce the impact "
            "of potential security vulnerabilities."
)
def last_user_is_root():
    """
    Detects when the final USER instruction in the Dockerfile is set to 'root'.

    This check applies to the final build stage in multi-stage builds.
    """
    return final_stage_has(
        instruction=instruction(type="USER", user_name="root")
    )
