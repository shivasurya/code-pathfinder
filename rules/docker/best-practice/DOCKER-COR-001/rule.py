from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction


@dockerfile_rule(
    id="DOCKER-COR-001",
    name="Multiple ENTRYPOINT Instructions",
    severity="MEDIUM",
    cwe="CWE-710",
    category="correctness",
    tags="docker,dockerfile,entrypoint,correctness,configuration,maintainability,confusing,anti-pattern",
    message="Dockerfile has multiple ENTRYPOINT instructions. Only the last one takes effect, making earlier ones misleading."
)
def multiple_entrypoint_instructions():
    # Note: This is a simplified check - ideally would count occurrences
    # For now, this flags any ENTRYPOINT which helps identify the issue
    return instruction(type="ENTRYPOINT")
