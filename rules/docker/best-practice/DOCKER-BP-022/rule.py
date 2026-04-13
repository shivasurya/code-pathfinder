from codepathfinder.container_decorators import dockerfile_rule
from codepathfinder.container_matchers import missing


@dockerfile_rule(
    id="DOCKER-BP-022",
    name="Missing HEALTHCHECK Instruction",
    severity="LOW",
    cwe="CWE-710",
    category="best-practice",
    tags="docker,dockerfile,healthcheck,monitoring,observability,orchestration,kubernetes,reliability,best-practice,availability",
    message="No HEALTHCHECK instruction. Container health cannot be monitored by orchestrators, reducing reliability and observability."
)
def missing_healthcheck():
    """
    Detects missing HEALTHCHECK instruction.

    Health checks allow Docker, Kubernetes, and other orchestrators to
    monitor application health and automatically restart failing containers,
    significantly improving availability.
    """
    return missing(instruction="HEALTHCHECK")
