from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction
from rules.container_combinators import any_of


@dockerfile_rule(
    id="DOCKER-SEC-007",
    name="Sudo Usage in Dockerfile",
    severity="MEDIUM",
    cwe="CWE-250",
    category="security",
    tags="docker,dockerfile,sudo,security,privilege-escalation,anti-pattern,best-practice,user,root,unnecessary",
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
