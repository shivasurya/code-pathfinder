from codepathfinder.container_decorators import dockerfile_rule
from codepathfinder.container_matchers import instruction
from codepathfinder.container_combinators import any_of


@dockerfile_rule(
    id="DOCKER-BP-016",
    name="Prefer JSON Notation for CMD/ENTRYPOINT",
    severity="LOW",
    cwe="CWE-710",
    category="best-practice",
    tags="docker,dockerfile,cmd,entrypoint,exec-form,json,signal-handling,best-practice,process-management,pid1",
    message="Use JSON notation (exec form) for CMD/ENTRYPOINT for proper signal handling."
)
def prefer_json_notation():
    return any_of(
        instruction(type="CMD", command_form="shell"),
        instruction(type="ENTRYPOINT", command_form="shell")
    )
