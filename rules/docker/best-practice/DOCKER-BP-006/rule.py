from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction
from rules.container_combinators import any_of


@dockerfile_rule(
    id="DOCKER-BP-006",
    name="Avoid apt-get upgrade",
    severity="MEDIUM",
    cwe="CWE-710",
    category="best-practice",
    tags="docker,dockerfile,apt-get,upgrade,package-manager,ubuntu,debian,reproducibility,best-practice,anti-pattern,build",
    message="Avoid apt-get upgrade in Dockerfiles. Use specific base image versions instead."
)
def avoid_apt_get_upgrade():
    return any_of(
        instruction(type="RUN", contains="apt-get upgrade"),
        instruction(type="RUN", contains="apt-get dist-upgrade")
    )
