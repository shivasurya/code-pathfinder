from codepathfinder.container_decorators import dockerfile_rule
from codepathfinder.container_matchers import instruction
from codepathfinder.container_combinators import all_of


@dockerfile_rule(
    id="DOCKER-BP-005",
    name="apt-get Without --no-install-recommends",
    severity="LOW",
    cwe="CWE-710",
    category="best-practice",
    tags="docker,dockerfile,apt-get,package-manager,ubuntu,debian,optimization,image-size,best-practice,bloat,attack-surface",
    message="apt-get install without --no-install-recommends. This installs unnecessary packages, increasing image size and attack surface."
)
def apt_without_no_recommends():
    """
    Detects apt-get install without --no-install-recommends flag.

    Without this flag, apt installs "recommended" packages which are
    often not needed and bloat the image by 30-50%.
    """
    return all_of(
        instruction(type="RUN", contains="apt-get install"),
        instruction(type="RUN", not_contains="--no-install-recommends")
    )
