from codepathfinder.container_decorators import dockerfile_rule
from rules.container_matchers import instruction


@dockerfile_rule(
    id="DOCKER-BP-011",
    name="Prefer COPY Over ADD",
    severity="LOW",
    cwe="CWE-710",
    category="best-practice",
    tags="docker,dockerfile,copy,add,file-operations,best-practice,transparency,predictability,anti-pattern",
    message="Use COPY instead of ADD for simple file operations. ADD has implicit behavior that can be surprising."
)
def prefer_copy_over_add():
    return instruction(type="ADD")
