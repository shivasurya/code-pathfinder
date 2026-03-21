from rules.container_decorators import compose_rule
from rules.container_matchers import service_has


@compose_rule(
    id="COMPOSE-SEC-001",
    name="Service Running in Privileged Mode",
    severity="CRITICAL",
    cwe="CWE-250",
    category="security",
    tags="docker-compose,compose,service,privileged,security,privilege-escalation,container-escape,capabilities,critical,host-access",
    message="Service is running in privileged mode. This grants container equivalent of root capabilities on the host machine. Can lead to container escapes and privilege escalation."
)
def privileged_service():
    """
    Detects services with privileged: true.

    Privileged mode disables almost all container isolation, giving
    the container nearly all capabilities of the host. This is extremely
    dangerous and should be avoided except in very rare circumstances.
    """
    return service_has(
        key="privileged",
        equals=True
    )
