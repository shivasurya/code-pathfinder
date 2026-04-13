from codepathfinder.container_decorators import compose_rule
from codepathfinder.container_matchers import service_has


@compose_rule(
    id="COMPOSE-SEC-008",
    name="Dangerous Capability Added",
    severity="HIGH",
    cwe="CWE-250",
    category="security",
    tags="docker-compose,compose,capabilities,cap-add,security,privilege-escalation,container-escape,linux,kernel",
    message="Service adds dangerous capability. These capabilities can be used for container escape or privilege escalation."
)
def dangerous_capabilities():
    """
    Detects services with dangerous capabilities.

    Capabilities like SYS_ADMIN, SYS_MODULE, and SYS_PTRACE provide
    near-root powers and can be exploited for container escape.
    """
    return service_has(
        key="cap_add",
        contains_any=[
            "SYS_ADMIN",
            "NET_ADMIN",
            "SYS_PTRACE",
            "SYS_MODULE",
            "DAC_READ_SEARCH",
            "ALL"
        ]
    )
