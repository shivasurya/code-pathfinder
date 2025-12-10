"""DOCKER-BP-022: Nonsensical Command (cd in same RUN)"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction
from rules.container_combinators import any_of


@dockerfile_rule(
    id="DOCKER-BP-030",
    name="Nonsensical Command",
    severity="LOW",
    category="best-practice",
    message="RUN command uses 'cd' which doesn't persist. Use WORKDIR instead."
)
def nonsensical_command():
    return any_of(
        instruction(type="RUN", regex=r";\s*cd\s+"),
        instruction(type="RUN", regex=r"&&\s*cd\s+")
    )
