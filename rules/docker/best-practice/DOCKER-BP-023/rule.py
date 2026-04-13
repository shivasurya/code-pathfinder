from codepathfinder.container_decorators import dockerfile_rule
from codepathfinder.container_matchers import instruction
from codepathfinder.container_combinators import all_of


@dockerfile_rule(
    id="DOCKER-BP-023",
    name="Prefer apt-get over apt",
    severity="LOW",
    cwe="CWE-710",
    category="best-practice",
    tags="docker,dockerfile,apt,apt-get,package-manager,ubuntu,debian,scripting,stability,reproducibility,best-practice",
    message="Use apt-get instead of apt for better script stability in Dockerfiles."
)
def prefer_apt_get():
    return all_of(
        instruction(type="RUN", regex=r"\bapt\s+install"),
        instruction(type="RUN", not_contains="apt-get")
    )
