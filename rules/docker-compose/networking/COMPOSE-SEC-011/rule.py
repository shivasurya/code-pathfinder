from codepathfinder.container_decorators import compose_rule
from codepathfinder.container_matchers import service_missing


@compose_rule(
    id="COMPOSE-SEC-011",
    name="Missing no-new-privileges Security Option",
    severity="MEDIUM",
    cwe="CWE-732",
    category="security",
    tags="docker-compose,compose,no-new-privileges,security,setuid,privilege-escalation,hardening,capabilities",
    message="Service does not have 'no-new-privileges:true' in security_opt. This allows "
            "processes to gain additional privileges via setuid/setgid binaries, which can be "
            "exploited for privilege escalation attacks."
)
def no_new_privileges():
    """
    Detects services missing the no-new-privileges security option.

    This check looks for services that either:
    1. Have no security_opt defined
    2. Have security_opt but don't include no-new-privileges:true
    """
    return service_missing(
        key="security_opt",
        value_contains="no-new-privileges:true"
    )
