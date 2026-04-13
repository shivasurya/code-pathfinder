from codepathfinder.container_decorators import dockerfile_rule
from codepathfinder.container_matchers import instruction
from codepathfinder.container_combinators import all_of


@dockerfile_rule(
    id="DOCKER-BP-014",
    name="Remove apt Package Lists",
    severity="LOW",
    cwe="CWE-710",
    category="best-practice",
    tags="docker,dockerfile,apt-get,package-manager,ubuntu,debian,cache,cleanup,image-size,optimization,best-practice",
    message="apt-get install without removing /var/lib/apt/lists/*. This wastes image space."
)
def remove_package_lists():
    return all_of(
        instruction(type="RUN", contains="apt-get install"),
        instruction(type="RUN", not_contains="/var/lib/apt/lists/")
    )
