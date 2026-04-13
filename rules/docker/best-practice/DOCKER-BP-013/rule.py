from codepathfinder.container_decorators import dockerfile_rule
from codepathfinder.container_matchers import instruction
from codepathfinder.container_combinators import all_of


@dockerfile_rule(
    id="DOCKER-BP-013",
    name="Missing dnf clean all",
    severity="LOW",
    cwe="CWE-710",
    category="best-practice",
    tags="docker,dockerfile,dnf,package-manager,fedora,rhel,cache,cleanup,image-size,optimization,best-practice",
    message="RUN uses 'dnf install' without 'dnf clean all'. This increases image size."
)
def missing_dnf_clean_all():
    return all_of(
        instruction(type="RUN", contains="dnf install"),
        instruction(type="RUN", not_contains="dnf clean all")
    )
