from codepathfinder.container_decorators import dockerfile_rule
from rules.container_matchers import instruction
from rules.container_combinators import all_of


@dockerfile_rule(
    id="DOCKER-BP-012",
    name="Missing yum clean all",
    severity="LOW",
    cwe="CWE-710",
    category="best-practice",
    tags="docker,dockerfile,yum,package-manager,centos,rhel,cache,cleanup,image-size,optimization,best-practice",
    message="RUN instruction uses 'yum install' without 'yum clean all'. This leaves package cache and increases image size."
)
def missing_yum_clean_all():
    return all_of(
        instruction(type="RUN", contains="yum install"),
        instruction(type="RUN", not_contains="yum clean all")
    )
