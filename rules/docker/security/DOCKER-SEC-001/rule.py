from rules.container_decorators import dockerfile_rule
from rules.container_matchers import missing


@dockerfile_rule(
    id="DOCKER-SEC-001",
    name="Container Running as Root - Missing USER",
    severity="HIGH",
    cwe="CWE-250",
    category="security",
    tags="docker,dockerfile,container,security,privilege-escalation,root,user,best-practice,hardening,principle-of-least-privilege",
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
