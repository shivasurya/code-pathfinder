from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction
from rules.container_combinators import all_of


@dockerfile_rule(
    id="DOCKER-BP-007",
    name="apk add Without --no-cache",
    severity="LOW",
    cwe="CWE-710",
    category="best-practice",
    tags="docker,dockerfile,apk,package-manager,alpine,cache,optimization,image-size,best-practice,linux",
    message="apk add without --no-cache. Package cache remains in image, increasing size by 2-5 MB."
)
def apk_without_no_cache():
    """
    Detects apk add without --no-cache flag for Alpine images.

    The --no-cache flag prevents package cache from being stored
    in the image, reducing size by 20-30% for Alpine-based images.
    """
    return all_of(
        instruction(type="RUN", contains="apk add"),
        instruction(type="RUN", not_contains="--no-cache")
    )
