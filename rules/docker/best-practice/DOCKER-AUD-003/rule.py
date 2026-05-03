from codepathfinder.container_decorators import dockerfile_rule
from codepathfinder.container_matchers import instruction


@dockerfile_rule(
    id="DOCKER-AUD-003",
    name="Privileged Port Exposed",
    severity="MEDIUM",
    cwe="CWE-250",
    category="audit",
    tags="docker,dockerfile,port,expose,privileged,root,security,unix,networking,capabilities,best-practice",
    message="Exposing port below 1024 typically requires root privileges to bind. Consider using non-privileged ports (>1024) with port mapping or granting CAP_NET_BIND_SERVICE capability."
)
def privileged_port():
    """
    Detects exposure of privileged ports (1-1023).

    Binding to privileged ports requires root privileges or CAP_NET_BIND_SERVICE
    capability, which conflicts with running containers as non-root users.
    """
    return instruction(
        type="EXPOSE",
        port_less_than=1024
    )
