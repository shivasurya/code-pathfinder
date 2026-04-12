from codepathfinder.container_decorators import compose_rule
from rules.container_matchers import service_has


@compose_rule(
    id="COMPOSE-SEC-012",
    name="SELinux Separation Disabled",
    severity="MEDIUM",
    cwe="CWE-732",
    category="security",
    tags="docker-compose,compose,selinux,security,mac,mandatory-access-control,isolation,hardening,rhel",
    message="Service has 'label:disable' in security_opt, which disables SELinux mandatory "
            "access control. This removes a critical security layer and increases the impact "
            "of container compromises. Remove label:disable or use custom SELinux labels instead."
)
def selinux_disabled():
    """
    Detects services that explicitly disable SELinux separation.

    Matches services with security_opt containing:
    - label:disable
    - label=disable
    """
    return service_has(
        key="security_opt",
        contains="label:disable"
    )
