from codepathfinder.container_decorators import dockerfile_rule
from rules.container_matchers import instruction


@dockerfile_rule(
    id="DOCKER-COR-003",
    name="Multiple CMD Instructions",
    severity="MEDIUM",
    cwe="CWE-710",
    category="correctness",
    tags="docker,dockerfile,cmd,correctness,configuration,maintainability,confusing,anti-pattern",
    message="Multiple CMD instructions detected. Only the last one takes effect."
)
def multiple_cmd_instructions():
    return instruction(type="CMD")
