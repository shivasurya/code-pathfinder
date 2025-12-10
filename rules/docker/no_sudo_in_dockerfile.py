"""
DOCKER-SEC-007: Sudo Usage in Dockerfile

Security Impact: MEDIUM
CWE: CWE-250 (Execution with Unnecessary Privileges)

DESCRIPTION:
This rule detects the use of 'sudo' in RUN instructions within a Dockerfile.
Using sudo in Docker containers is an anti-pattern that indicates confusion about
Docker's privilege model and can introduce security vulnerabilities.

SECURITY IMPLICATIONS:
Using sudo in Dockerfiles is problematic because:

1. **Unnecessary Complexity**: Docker containers already run commands as root by default
   during build time, making sudo redundant and confusing.

2. **False Sense of Security**: Developers may assume sudo provides security isolation,
   when in reality it adds no protection in a container context.

3. **Privilege Escalation Path**: If sudo is installed and configured in the final image,
   it provides an easy privilege escalation mechanism if an attacker gains access.

4. **Attack Surface**: sudo binary itself has had security vulnerabilities (CVE-2021-3156
   "Baron Samedit") that can be exploited if present in the container.

WHY SUDO DOESN'T MAKE SENSE IN DOCKER:
```dockerfile
# WRONG: Redundant sudo during build (already root)
RUN sudo apt-get update

# CORRECT: Just run the command (build runs as root)
RUN apt-get update

# WRONG: Using sudo to run as different user
RUN sudo -u appuser /app/script.sh

# CORRECT: Use USER instruction instead
USER appuser
RUN /app/script.sh
```

VULNERABLE EXAMPLE:
```dockerfile
FROM ubuntu:latest

# Installing sudo (unnecessary and dangerous)
RUN apt-get update && apt-get install -y sudo

# Using sudo in RUN (redundant - already root)
RUN sudo apt-get install -y nginx

# Creating sudoers file (privilege escalation risk)
RUN echo "appuser ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers

USER appuser
# Now attacker with access can easily become root
CMD ["sudo", "/bin/bash"]
```

SECURE ALTERNATIVES:
```dockerfile
FROM ubuntu:latest

# Install packages without sudo (we're already root)
RUN apt-get update && apt-get install -y nginx \\
    && apt-get clean

# Create non-root user
RUN useradd -r -s /bin/false appuser

# Switch to non-root for runtime
USER appuser

# Run application as non-root
CMD ["nginx", "-g", "daemon off;"]
```

WHEN YOU THINK YOU NEED SUDO:

**Scenario 1: "I need to install packages"**
- You're already root during build - no sudo needed
- Just use `RUN apt-get install package`

**Scenario 2: "I need to run as different user"**
- Use `USER username` instruction instead
- Much cleaner and more explicit

**Scenario 3: "Runtime needs elevated permissions"**
- This is a design smell - containers should run as non-root
- Reconsider architecture or use capabilities instead

**Scenario 4: "CI/CD needs to run Docker commands"**
- Don't mount Docker socket - use Kaniko, BuildKit remote, or DooD proxy
- Never rely on sudo for container orchestration

REMEDIATION:
1. Remove all `sudo` commands from RUN instructions
2. Remove sudo package from installed dependencies
3. Use USER instruction for privilege changes
4. Run containers as non-root users
5. Use Linux capabilities if elevated permissions are truly needed

REFERENCES:
- CWE-250: Execution with Unnecessary Privileges
- https://docs.docker.com/develop/develop-images/dockerfile_best-practices/#user
- Docker Security Best Practices
- CVE-2021-3156 (sudo vulnerability)
"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction
from rules.container_combinators import any_of


@dockerfile_rule(
    id="DOCKER-SEC-007",
    name="Sudo Usage in Dockerfile",
    severity="MEDIUM",
    cwe="CWE-250",
    category="security",
    message="Dockerfile uses 'sudo' in RUN instructions. This is unnecessary during "
            "build (already root) and increases security risk if sudo remains in the "
            "final image. Use USER instruction for privilege changes instead."
)
def no_sudo_in_dockerfile():
    """
    Detects usage of sudo in RUN instructions.

    Matches patterns like:
    - RUN sudo apt-get install
    - RUN sudo -u user command
    - RUN sudo command

    """
    return any_of(
        instruction(type="RUN", contains="sudo "),
        instruction(type="RUN", regex=r"sudo\s+"),
        instruction(type="RUN", contains="sudo\n"),
        instruction(type="RUN", contains="sudo;"),
        instruction(type="RUN", contains="sudo&&"),
        instruction(type="RUN", contains="sudo||")
    )
