"""DOCKER-BP-025: Missing -y flag for yum"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction
from rules.container_combinators import all_of


@dockerfile_rule(
    id="DOCKER-BP-025",
    name="Missing -y flag for yum",
    severity="LOW",
    category="best-practice",
    message="yum install without -y flag. Add -y for non-interactive builds."
)
def missing_yum_assume_yes():
    return all_of(
        instruction(type="RUN", contains="yum install"),
        instruction(type="RUN", not_contains="-y")
    )
