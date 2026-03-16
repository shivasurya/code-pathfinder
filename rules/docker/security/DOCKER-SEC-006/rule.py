from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction
from rules.container_combinators import any_of


@dockerfile_rule(
    id="DOCKER-SEC-006",
    name="Docker Socket Mounted as Volume",
    severity="CRITICAL",
    cwe="CWE-250",
    category="security",
    tags="docker,dockerfile,docker-socket,volume,security,privilege-escalation,container-escape,daemon,host-access,critical",
    message="Dockerfile mounts Docker socket. This gives the container full control over the host Docker daemon, equivalent to root access."
)
def docker_socket_in_volume():
    """
    Detects VOLUME instructions that include Docker socket paths.

    Mounting the Docker socket inside a container gives it full control
    over the Docker daemon, allowing it to create privileged containers,
    access host filesystem, and effectively become root on the host.
    """
    return any_of(
        instruction(type="VOLUME", contains="/var/run/docker.sock"),
        instruction(type="VOLUME", contains="/run/docker.sock"),
        instruction(type="VOLUME", contains="docker.sock")
    )
