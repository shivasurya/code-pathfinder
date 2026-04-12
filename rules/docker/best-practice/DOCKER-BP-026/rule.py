from codepathfinder.container_decorators import dockerfile_rule
from rules.container_matchers import instruction
from rules.container_combinators import all_of


@dockerfile_rule(
    id="DOCKER-BP-026",
    name="Missing -y flag for dnf",
    severity="LOW",
    cwe="CWE-710",
    category="best-practice",
    tags="docker,dockerfile,dnf,package-manager,fedora,rhel,automation,ci-cd,build,best-practice,non-interactive",
    message="dnf install without -y flag. Add -y for non-interactive builds."
)
def missing_dnf_assume_yes():
    return all_of(
        instruction(type="RUN", contains="dnf install"),
        instruction(type="RUN", not_contains="-y")
    )
