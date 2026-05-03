from codepathfinder.container_decorators import dockerfile_rule
from codepathfinder.container_matchers import instruction, missing
from codepathfinder.container_combinators import all_of, any_of


@dockerfile_rule(
    id="DOCKER-BP-017",
    name="Use WORKDIR Instead of cd",
    severity="LOW",
    cwe="CWE-710",
    category="best-practice",
    tags="docker,dockerfile,workdir,cd,directory,best-practice,maintainability,clarity,anti-pattern",
    message="Use WORKDIR instruction instead of 'cd' in RUN commands."
)
def use_workdir():
    return all_of(
        any_of(
            instruction(type="RUN", contains=" cd "),
            instruction(type="RUN", regex=r"\bcd\s+")
        ),
        missing(instruction="WORKDIR")
    )
