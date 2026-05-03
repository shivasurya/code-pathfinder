from codepathfinder.container_decorators import dockerfile_rule
from codepathfinder.container_matchers import instruction


@dockerfile_rule(
    id="DOCKER-BP-027",
    name="Avoid --platform Flag with FROM",
    severity="LOW",
    cwe="CWE-710",
    category="best-practice",
    tags="docker,dockerfile,from,platform,multi-arch,portability,buildx,architecture,best-practice",
    message="FROM with --platform flag reduces portability. Let Docker handle platform selection."
)
def avoid_platform_with_from():
    return instruction(type="FROM", contains="--platform")
