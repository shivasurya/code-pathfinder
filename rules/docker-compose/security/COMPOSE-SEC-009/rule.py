from codepathfinder.container_decorators import compose_rule
from rules.container_matchers import service_has


@compose_rule(
    id="COMPOSE-SEC-009",
    name="Using Host PID Mode",
    severity="HIGH",
    cwe="CWE-250",
    category="security",
    tags="docker-compose,compose,pid,host-pid,security,isolation,namespace,process,information-disclosure",
    message="Service uses host PID namespace. Container can see and potentially signal host processes."
)
def host_pid_mode():
    """
    Detects services using host PID namespace.

    Sharing the host PID namespace allows the container to view all
    host processes, send signals to them, and access sensitive process
    information via /proc.
    """
    return service_has(
        key="pid",
        equals="host"
    )
