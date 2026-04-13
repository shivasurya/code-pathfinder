from codepathfinder.container_decorators import compose_rule
from codepathfinder.container_matchers import service_has


@compose_rule(
    id="COMPOSE-SEC-002",
    name="Docker Socket Exposed to Container",
    severity="CRITICAL",
    cwe="CWE-250",
    category="security",
    tags="docker-compose,compose,docker-socket,volume,security,privilege-escalation,container-escape,daemon,critical,host-access",
    message="Service mounts Docker socket. The owner of this socket is root. Giving container access to it is equivalent to giving unrestricted root access to host."
)
def docker_socket_exposed():
    """
    Detects Docker socket mounted as volume.

    Mounting /var/run/docker.sock gives the container full control over
    the Docker daemon, allowing container escape, privilege escalation,
    and complete host compromise.
    """
    return service_has(
        key="volumes",
        contains_any=[
            "/var/run/docker.sock",
            "/run/docker.sock",
            "docker.sock"
        ]
    )
