from rules.container_decorators import compose_rule
from rules.container_matchers import service_has


@compose_rule(
    id="COMPOSE-SEC-007",
    name="Using Host Network Mode",
    severity="HIGH",
    cwe="CWE-250",
    category="security",
    tags="docker-compose,compose,network,host-network,security,isolation,networking,namespace,privilege-escalation",
    message="Service uses host network mode. Container shares host network stack, bypassing network isolation."
)
def host_network_mode():
    """
    Detects services using host network mode.

    Host network mode disables network namespace isolation, allowing
    the container to access all host network interfaces and localhost
    services, significantly increasing attack surface.
    """
    return service_has(
        key="network_mode",
        equals="host"
    )
