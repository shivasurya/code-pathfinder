from codepathfinder.container_decorators import dockerfile_rule
from codepathfinder.container_matchers import instruction
from codepathfinder.container_combinators import all_of


@dockerfile_rule(
    id="DOCKER-BP-008",
    name="pip install Without --no-cache-dir",
    severity="LOW",
    cwe="CWE-710",
    category="best-practice",
    tags="docker,dockerfile,pip,python,package-manager,cache,optimization,image-size,best-practice",
    message="pip install without --no-cache-dir. Pip cache remains in image, adding 50-200 MB depending on dependencies."
)
def pip_without_no_cache():
    """
    Detects pip install without --no-cache-dir flag.

    pip caches downloaded packages in /root/.cache/pip/ which can
    add 50-200 MB to images. Use --no-cache-dir or ENV PIP_NO_CACHE_DIR=1.
    """
    return all_of(
        instruction(type="RUN", contains="pip install"),
        instruction(type="RUN", not_contains="--no-cache-dir")
    )
