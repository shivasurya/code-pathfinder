"""
Dockerfile Security Rules for Code Pathfinder.

This module provides security analysis for Dockerfiles covering privilege
management, supply chain security, and best practices.

Rule Categories:
- DOCKER-SEC-*: Security vulnerabilities (CWE mappings)
- DOCKER-BP-*: Best practice violations
- DOCKER-AUD-*: Audit/informational findings
"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction, missing
from rules.container_combinators import all_of, any_of


@dockerfile_rule(
    id="DOCKER-SEC-001",
    name="Container Running as Root - Missing USER",
    severity="HIGH",
    cwe="CWE-250",
    category="security",
    message="Dockerfile does not specify USER instruction. Container will run as root by default, which increases the attack surface if the container is compromised."
)
def missing_user_instruction():
    """
    Detects Dockerfiles that do not specify a USER instruction.

    Running containers as root is a security risk because if an attacker
    gains access to the container, they have root privileges which can be
    used for privilege escalation or lateral movement.
    """
    return missing(instruction="USER")


@dockerfile_rule(
    id="DOCKER-SEC-005",
    name="Secret in Build Argument",
    severity="CRITICAL",
    cwe="CWE-538",
    category="security",
    message="Build argument name suggests it contains a secret. ARG values are visible in image history via 'docker history'."
)
def secret_in_build_arg():
    """
    Detects ARG instructions with names suggesting secrets.

    Build arguments are stored in the image layer history and can be
    retrieved by anyone with access to the image. Never pass secrets
    as build arguments.
    """
    return instruction(
        type="ARG",
        arg_name_regex=r"(?i)^.*(password|passwd|secret|token|key|apikey|api_key|auth|credential|cred|private|access_token|client_secret).*$"
    )


@dockerfile_rule(
    id="DOCKER-SEC-006",
    name="Docker Socket Mounted as Volume",
    severity="CRITICAL",
    cwe="CWE-250",
    category="security",
    message="Dockerfile mounts Docker socket. This gives the container full control over the host Docker daemon, equivalent to root access."
)
def docker_socket_in_volume():
    """
    Detects VOLUME instructions that include Docker socket paths.

    Mounting the Docker socket inside a container gives it full control
    over the Docker daemon, allowing it to create privileged containers,
    access host filesystem, and effectively become root on the host.
    """
    return any_of(
        instruction(type="VOLUME", contains="/var/run/docker.sock"),
        instruction(type="VOLUME", contains="/run/docker.sock"),
        instruction(type="VOLUME", contains="docker.sock")
    )


@dockerfile_rule(
    id="DOCKER-BP-001",
    name="Base Image Uses :latest Tag",
    severity="MEDIUM",
    category="best-practice",
    message="Base image uses ':latest' tag or no tag (defaults to latest). This makes builds non-reproducible.",
)
def using_latest_tag():
    """
    Detects FROM instructions using :latest or implicit latest tag.

    Using :latest leads to non-reproducible builds as the underlying
    image can change at any time. Always pin to specific versions.
    """
    return instruction(type="FROM", image_tag="latest")


@dockerfile_rule(
    id="DOCKER-BP-003",
    name="Deprecated MAINTAINER Instruction",
    severity="INFO",
    category="best-practice",
    message="MAINTAINER instruction is deprecated. Use LABEL instead.",
)
def maintainer_deprecated():
    """
    Detects usage of deprecated MAINTAINER instruction.

    The MAINTAINER instruction is deprecated in favor of LABEL.
    """
    return instruction(type="MAINTAINER")


@dockerfile_rule(
    id="DOCKER-BP-005",
    name="apt-get Without --no-install-recommends",
    severity="LOW",
    category="best-practice",
    message="apt-get install without --no-install-recommends. This installs unnecessary packages, increasing image size.",
)
def apt_without_no_recommends():
    """
    Detects apt-get install without --no-install-recommends flag.

    Without this flag, apt installs "recommended" packages which are
    often not needed and bloat the image.
    """
    return all_of(
        instruction(type="RUN", contains="apt-get install"),
        instruction(type="RUN", not_contains="--no-install-recommends")
    )


@dockerfile_rule(
    id="DOCKER-BP-007",
    name="apk add Without --no-cache",
    severity="LOW",
    category="best-practice",
    message="apk add without --no-cache. Package cache remains in image.",
)
def apk_without_no_cache():
    """
    Detects apk add without --no-cache flag for Alpine images.
    """
    return all_of(
        instruction(type="RUN", contains="apk add"),
        instruction(type="RUN", not_contains="--no-cache")
    )


@dockerfile_rule(
    id="DOCKER-BP-008",
    name="pip install Without --no-cache-dir",
    severity="LOW",
    category="best-practice",
    message="pip install without --no-cache-dir. Pip cache remains in image.",
)
def pip_without_no_cache():
    """
    Detects pip install without --no-cache-dir flag.
    """
    return all_of(
        instruction(type="RUN", contains="pip install"),
        instruction(type="RUN", not_contains="--no-cache-dir")
    )


@dockerfile_rule(
    id="DOCKER-BP-022",
    name="Missing HEALTHCHECK Instruction",
    severity="LOW",
    category="best-practice",
    message="No HEALTHCHECK instruction. Container health cannot be monitored by orchestrators.",
)
def missing_healthcheck():
    """
    Detects missing HEALTHCHECK instruction.
    """
    return missing(instruction="HEALTHCHECK")


@dockerfile_rule(
    id="DOCKER-AUD-003",
    name="Privileged Port Exposed",
    severity="MEDIUM",
    category="audit",
    message="Exposing port below 1024 requires root privileges to bind.",
)
def privileged_port():
    """
    Detects exposure of privileged ports.
    """
    return instruction(
        type="EXPOSE",
        port_less_than=1024
    )


# Rule registry for compilation
DOCKERFILE_RULES = [
    missing_user_instruction,
    secret_in_build_arg,
    docker_socket_in_volume,
    using_latest_tag,
    maintainer_deprecated,
    apt_without_no_recommends,
    apk_without_no_cache,
    pip_without_no_cache,
    missing_healthcheck,
    privileged_port,
]
