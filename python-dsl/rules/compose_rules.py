"""
Docker Compose Security Rules for Code Pathfinder.

This module provides security analysis for docker-compose YAML configuration
files, focusing on privilege escalation, security controls, and isolation.

Rule Categories:
- COMPOSE-SEC-*: Security vulnerabilities
"""

from rules.container_decorators import compose_rule
from rules.container_matchers import service_has, service_missing
from rules.container_combinators import any_of


@compose_rule(
    id="COMPOSE-SEC-001",
    name="Service Running in Privileged Mode",
    severity="CRITICAL",
    cwe="CWE-250",
    category="security",
    message="Service is running in privileged mode. This grants container equivalent of root capabilities on the host machine. Can lead to container escapes and privilege escalation.",
)
def privileged_service():
    """
    Detects services with privileged: true.

    Privileged mode disables almost all container isolation, giving
    the container nearly all capabilities of the host.
    """
    return service_has(
        key="privileged",
        equals=True
    )


@compose_rule(
    id="COMPOSE-SEC-002",
    name="Docker Socket Exposed to Container",
    severity="CRITICAL",
    cwe="CWE-250",
    category="security",
    message="Service mounts Docker socket. The owner of this socket is root. Giving container access to it is equivalent to giving unrestricted root access to host.",
)
def docker_socket_exposed():
    """
    Detects Docker socket mounted as volume.
    """
    return service_has(
        key="volumes",
        contains_any=[
            "/var/run/docker.sock",
            "/run/docker.sock",
            "docker.sock"
        ]
    )


@compose_rule(
    id="COMPOSE-SEC-003",
    name="Seccomp Confinement Disabled",
    severity="HIGH",
    cwe="CWE-284",
    category="security",
    message="Service disables seccomp profile. Container can use all system calls, increasing attack surface.",
)
def seccomp_disabled():
    """
    Detects services with seccomp disabled.

    Seccomp limits which system calls a container can make. Disabling
    it removes an important security layer.
    """
    return service_has(
        key="security_opt",
        contains="seccomp:unconfined"
    )


@compose_rule(
    id="COMPOSE-SEC-006",
    name="Container Filesystem is Writable",
    severity="LOW",
    cwe="CWE-732",
    category="security",
    message="Service has writable root filesystem. Consider making it read-only for better security.",
)
def writable_filesystem():
    """
    Detects services without read_only: true.
    """
    return service_missing(key="read_only")


@compose_rule(
    id="COMPOSE-SEC-007",
    name="Using Host Network Mode",
    severity="HIGH",
    cwe="CWE-250",
    category="security",
    message="Service uses host network mode. Container shares host network stack, bypassing network isolation.",
)
def host_network_mode():
    """
    Detects services using host network mode.
    """
    return service_has(
        key="network_mode",
        equals="host"
    )


@compose_rule(
    id="COMPOSE-SEC-008",
    name="Dangerous Capability Added",
    severity="HIGH",
    cwe="CWE-250",
    category="security",
    message="Service adds dangerous capability. These capabilities can be used for container escape or privilege escalation.",
)
def dangerous_capabilities():
    """
    Detects services with dangerous capabilities.
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


@compose_rule(
    id="COMPOSE-SEC-009",
    name="Using Host PID Mode",
    severity="HIGH",
    cwe="CWE-250",
    category="security",
    message="Service uses host PID namespace. Container can see and potentially signal host processes.",
)
def host_pid_mode():
    """
    Detects services using host PID namespace.
    """
    return service_has(
        key="pid",
        equals="host"
    )


@compose_rule(
    id="COMPOSE-SEC-010",
    name="Using Host IPC Mode",
    severity="MEDIUM",
    cwe="CWE-250",
    category="security",
    message="Service uses host IPC namespace. Container shares inter-process communication with host.",
)
def host_ipc_mode():
    """
    Detects services using host IPC namespace.
    """
    return service_has(
        key="ipc",
        equals="host"
    )


# Rule registry for compilation
COMPOSE_RULES = [
    privileged_service,
    docker_socket_exposed,
    seccomp_disabled,
    writable_filesystem,
    host_network_mode,
    dangerous_capabilities,
    host_pid_mode,
    host_ipc_mode,
]
