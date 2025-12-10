"""DOCKER-BP-008: Avoid yum update in Dockerfile"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction


@dockerfile_rule(
    id="DOCKER-BP-029",
    name="Avoid yum update",
    severity="MEDIUM",
    category="best-practice",
    message="Avoid 'yum update' in Dockerfiles. Use specific base image versions for reproducible builds."
)
def avoid_yum_update():
    return instruction(type="RUN", contains="yum update")
