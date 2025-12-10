"""DOCKER-BP-009: Avoid dnf update in Dockerfile"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction


@dockerfile_rule(
    id="DOCKER-BP-009",
    name="Avoid dnf update",
    severity="MEDIUM",
    category="best-practice",
    message="Avoid 'dnf update' in Dockerfiles. Use specific base image versions for reproducible builds."
)
def avoid_dnf_update():
    return instruction(type="RUN", contains="dnf update")
