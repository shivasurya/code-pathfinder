"""DOCKER-BP-007: Avoid apk upgrade in Dockerfile"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction


@dockerfile_rule(
    id="DOCKER-BP-028",
    name="Avoid apk upgrade",
    severity="MEDIUM",
    category="best-practice",
    message="Avoid 'apk upgrade' in Dockerfiles. Use specific base image versions instead for reproducible builds."
)
def avoid_apk_upgrade():
    return instruction(type="RUN", contains="apk upgrade")
