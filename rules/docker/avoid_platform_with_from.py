"""DOCKER-BP-027: Avoid --platform Flag with FROM"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction


@dockerfile_rule(
    id="DOCKER-BP-027",
    name="Avoid --platform Flag with FROM",
    severity="LOW",
    category="best-practice",
    message="FROM with --platform flag reduces portability. Let Docker handle platform selection."
)
def avoid_platform_with_from():
    return instruction(type="FROM", contains="--platform")
