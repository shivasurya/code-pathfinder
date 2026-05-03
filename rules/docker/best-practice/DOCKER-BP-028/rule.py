from codepathfinder.container_decorators import dockerfile_rule
from codepathfinder.container_matchers import instruction


@dockerfile_rule(
    id="DOCKER-BP-028",
    name="Avoid apk upgrade",
    severity="MEDIUM",
    cwe="CWE-710",
    category="best-practice",
    tags="docker,dockerfile,apk,package-manager,alpine,upgrade,reproducibility,best-practice,anti-pattern",
    message="Avoid 'apk upgrade' in Dockerfiles. Use specific base image versions instead for reproducible builds."
)
def avoid_apk_upgrade():
    return instruction(type="RUN", contains="apk upgrade")
