from codepathfinder.container_decorators import dockerfile_rule
from codepathfinder.container_matchers import instruction
from codepathfinder.container_combinators import any_of


@dockerfile_rule(
    id="DOCKER-COR-002",
    name="Invalid Port Number",
    severity="HIGH",
    cwe="CWE-20",
    category="correctness",
    tags="docker,dockerfile,port,expose,validation,input-validation,correctness,networking,configuration",
    message="EXPOSE instruction has invalid port number. Valid ports are 1-65535."
)
def invalid_port():
    return any_of(
        instruction(type="EXPOSE", port_less_than=1),
        instruction(type="EXPOSE", port_greater_than=65535)
    )
