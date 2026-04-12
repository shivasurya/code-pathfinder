from codepathfinder.container_decorators import compose_rule
from rules.container_matchers import service_missing


@compose_rule(
    id="COMPOSE-SEC-006",
    name="Container Filesystem is Writable",
    severity="LOW",
    cwe="CWE-732",
    category="security",
    tags="docker-compose,compose,filesystem,read-only,security,immutability,malware-prevention,hardening,best-practice",
    message="Service has writable root filesystem. Consider making it read-only for better security."
)
def writable_filesystem():
    """
    Detects services without read_only: true.

    Read-only filesystems prevent attackers from modifying binaries,
    installing malware, or persisting backdoors.
    """
    return service_missing(key="read_only")
