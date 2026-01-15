"""DOCKER-BP-020: Missing zypper clean"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction
from rules.container_combinators import all_of


@dockerfile_rule(
    id="DOCKER-BP-020",
    name="Missing zypper clean",
    severity="LOW",
    category="best-practice",
    message="RUN uses 'zypper install' without 'zypper clean'. This increases image size."
)
def missing_zypper_clean():
    return all_of(
        instruction(type="RUN", contains="zypper install"),
        instruction(type="RUN", not_contains="zypper clean")
    )
