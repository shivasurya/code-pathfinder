"""DOCKER-BP-024: Install Only One of wget or curl"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction
from rules.container_combinators import all_of


@dockerfile_rule(
    id="DOCKER-BP-024",
    name="Install Only One of wget or curl",
    severity="LOW",
    category="best-practice",
    message="Installing both wget and curl wastes space. Choose one tool for downloads."
)
def use_either_wget_or_curl():
    return all_of(
        instruction(type="RUN", contains="wget"),
        instruction(type="RUN", contains="curl")
    )
