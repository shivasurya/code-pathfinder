"""
DOCKER-SEC-001: Container Running as Root - Missing USER Instruction

Security Impact: HIGH
CWE: CWE-250 (Execution with Unnecessary Privileges)

DESCRIPTION:
This rule detects Dockerfiles that do not specify a USER instruction, causing
containers to run with root privileges by default. Running containers as root
significantly increases the attack surface and potential impact of a container
compromise.

SECURITY IMPLICATIONS:
When a container runs as root (UID 0), any process inside the container has
unrestricted access to the container's filesystem and resources. If an attacker
compromises the application running inside the container, they gain root-level
capabilities which can be exploited for:

1. Container Escape: Using kernel exploits or misconfigurations to break out
   of the container and gain access to the host system
2. Privilege Escalation: Exploiting volume mounts or Docker socket access
3. Lateral Movement: Using elevated privileges to access other containers or
   network resources
4. Data Exfiltration: Reading sensitive files that may be mounted or accessible
5. Malware Installation: Installing persistent backdoors or cryptocurrency miners

VULNERABLE EXAMPLE:
```dockerfile
FROM ubuntu:22.04

# No USER instruction - container runs as root
RUN apt-get update && apt-get install -y nginx
COPY app /app
CMD ["nginx", "-g", "daemon off;"]
```

In this example, nginx runs as root, which is unnecessary and dangerous. If nginx
has a vulnerability, an attacker gains root access immediately.

SECURE EXAMPLE:
```dockerfile
FROM ubuntu:22.04

# Create non-root user with specific UID/GID
RUN groupadd -r appuser -g 999 && \
    useradd -r -u 999 -g appuser appuser

# Install dependencies as root
RUN apt-get update && apt-get install -y nginx

# Switch to non-root user
USER appuser

# Copy application files (now owned by appuser)
COPY --chown=appuser:appuser app /app

CMD ["nginx", "-g", "daemon off;"]
```

BEST PRACTICES:
1. Always create a dedicated user with a specific UID/GID
2. Use numeric UIDs (e.g., USER 999) rather than names for portability
3. Switch to non-root user BEFORE copying application files
4. Use --chown flag with COPY/ADD to set correct ownership
5. Never use UID 0 (root) for application processes
6. Consider using distroless or minimal base images

EXCEPTIONS:
- Init containers that need to perform system-level setup
- Containers explicitly designed to run privileged operations
- Build stages (multi-stage builds) that only run during image creation

REMEDIATION:
Add a USER instruction after installing dependencies but before starting the
application. Create a dedicated user with minimal privileges:

```dockerfile
RUN useradd -r -u 1000 -g appgroup appuser
USER appuser
```

REFERENCES:
- CWE-250: Execution with Unnecessary Privileges
- NIST SP 800-190: Application Container Security Guide
- Docker Security Best Practices
- Principle of Least Privilege (PoLP)
"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import missing


@dockerfile_rule(
    id="DOCKER-SEC-001",
    name="Container Running as Root - Missing USER",
    severity="HIGH",
    cwe="CWE-250",
    category="security",
    message="Dockerfile does not specify USER instruction. Container will run as root by default, which increases the attack surface if the container is compromised."
)
def missing_user_instruction():
    """
    Detects Dockerfiles that do not specify a USER instruction.

    Running containers as root is a security risk because if an attacker
    gains access to the container, they have root privileges which can be
    used for privilege escalation or lateral movement.
    """
    return missing(instruction="USER")

# Output JSON IR for Go executor
if __name__ == "__main__":
    import json
    import sys
    sys.path.insert(0, '/Users/shiva/src/shivasurya/code-pathfinder/python-dsl')

    from rules import container_decorators, container_ir

    # Get registered rules and convert to JSON IR
    json_ir = container_ir.compile_all_rules()

    # Output complete structure with both dockerfile and compose arrays
    output = {
        "dockerfile": json_ir.get("dockerfile", []),
        "compose": json_ir.get("compose", [])
    }
    print(json.dumps(output))
