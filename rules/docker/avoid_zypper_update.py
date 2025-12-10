"""DOCKER-BP-019: Avoid zypper update in Dockerfile"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction


@dockerfile_rule(
    id="DOCKER-BP-019",
    name="Avoid zypper update",
    severity="MEDIUM",
    category="best-practice",
    message="Avoid 'zypper update' in Dockerfiles. Use specific base image versions for reproducible builds."
)
def avoid_zypper_update():
    return instruction(type="RUN", contains="zypper update")
