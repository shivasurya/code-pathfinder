from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction
from rules.container_combinators import any_of


@dockerfile_rule(
    id="DOCKER-BP-015",
    name="Missing Image Version",
    severity="HIGH",
    cwe="CWE-1188",
    category="best-practice",
    tags="docker,dockerfile,from,image,tag,version,latest,reproducibility,best-practice,supply-chain,dependency-management",
    message="FROM instruction uses 'latest' tag or no tag. Specify explicit versions for reproducible builds."
)
def missing_image_version():
    return any_of(
        instruction(type="FROM", image_tag="latest"),
        instruction(type="FROM", missing_digest=True)
    )
