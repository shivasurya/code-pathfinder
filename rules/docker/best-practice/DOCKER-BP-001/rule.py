from codepathfinder.container_decorators import dockerfile_rule
from codepathfinder.container_matchers import instruction


@dockerfile_rule(
    id="DOCKER-BP-001",
    name="Base Image Uses :latest Tag",
    severity="MEDIUM",
    cwe="CWE-1188",
    category="best-practice",
    tags="docker,dockerfile,from,image,tag,version,latest,reproducibility,best-practice,supply-chain,immutability",
    message="Base image uses ':latest' tag or no tag (defaults to latest). This makes builds non-reproducible."
)
def using_latest_tag():
    """
    Detects FROM instructions using :latest or implicit latest tag.

    Using :latest leads to non-reproducible builds as the underlying
    image can change at any time. Always pin to specific versions.
    """
    return instruction(type="FROM", image_tag="latest")
