from codepathfinder.container_decorators import compose_rule
from codepathfinder.container_matchers import service_has


@compose_rule(
    id="COMPOSE-SEC-010",
    name="Using Host IPC Mode",
    severity="MEDIUM",
    cwe="CWE-250",
    category="security",
    tags="docker-compose,compose,ipc,host-ipc,security,isolation,namespace,shared-memory,information-disclosure",
    message="Service uses host IPC namespace. Container shares inter-process communication with host."
)
def host_ipc_mode():
    """
    Detects services using host IPC namespace.

    Sharing the host IPC namespace allows the container to access
    shared memory segments, semaphores, and message queues from
    host processes, potentially exposing sensitive data.
    """
    return service_has(
        key="ipc",
        equals="host"
    )
