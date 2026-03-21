from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction


@dockerfile_rule(
    id="DOCKER-AUD-001",
    name="Dockerfile Source Not Pinned",
    severity="LOW",
    cwe="CWE-1188",
    category="audit",
    tags="docker,dockerfile,from,digest,sha256,immutability,supply-chain,reproducibility,audit,security",
    message="FROM instruction without digest pinning. Consider using @sha256:... for immutable builds."
)
def dockerfile_source_not_pinned():
    return instruction(type="FROM", missing_digest=True)
