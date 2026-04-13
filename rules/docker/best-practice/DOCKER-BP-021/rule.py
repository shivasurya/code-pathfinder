from codepathfinder.container_decorators import dockerfile_rule
from codepathfinder.container_matchers import instruction
from codepathfinder.container_combinators import all_of


@dockerfile_rule(
    id="DOCKER-BP-021",
    name="Missing -y flag for apt-get",
    severity="LOW",
    cwe="CWE-710",
    category="best-practice",
    tags="docker,dockerfile,apt-get,package-manager,automation,ci-cd,build,ubuntu,debian,best-practice,non-interactive",
    message="apt-get install without -y flag. Add -y or --yes for non-interactive builds."
)
def missing_apt_assume_yes():
    return all_of(
        instruction(type="RUN", contains="apt-get install"),
        instruction(type="RUN", not_contains="-y"),
        instruction(type="RUN", not_contains="--yes")
    )
