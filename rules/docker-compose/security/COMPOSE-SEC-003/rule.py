from codepathfinder.container_decorators import compose_rule
from codepathfinder.container_matchers import service_has


@compose_rule(
    id="COMPOSE-SEC-003",
    name="Seccomp Confinement Disabled",
    severity="HIGH",
    cwe="CWE-284",
    category="security",
    tags="docker-compose,compose,seccomp,security,syscall,kernel,confinement,isolation,attack-surface",
    message="Service disables seccomp profile. Container can use all system calls, increasing attack surface."
)
def seccomp_disabled():
    """
    Detects services with seccomp disabled.

    Seccomp limits which system calls a container can make. Disabling
    it removes an important security layer and allows dangerous operations
    like kernel module loading and process injection.
    """
    return service_has(
        key="security_opt",
        contains="seccomp:unconfined"
    )
