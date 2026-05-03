from codepathfinder.container_decorators import dockerfile_rule
from codepathfinder.container_matchers import instruction


@dockerfile_rule(
    id="DOCKER-BP-018",
    name="Use Absolute Path in WORKDIR",
    severity="LOW",
    cwe="CWE-710",
    category="best-practice",
    tags="docker,dockerfile,workdir,path,absolute-path,best-practice,clarity,maintainability,filesystem",
    message="WORKDIR should use absolute paths starting with /."
)
def use_absolute_workdir():
    return instruction(type="WORKDIR", workdir_not_absolute=True)
