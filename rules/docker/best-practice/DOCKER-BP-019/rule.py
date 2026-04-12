from codepathfinder.container_decorators import dockerfile_rule
from rules.container_matchers import instruction


@dockerfile_rule(
    id="DOCKER-BP-019",
    name="Avoid zypper update",
    severity="MEDIUM",
    cwe="CWE-710",
    category="best-practice",
    tags="docker,dockerfile,zypper,package-manager,opensuse,suse,update,reproducibility,best-practice,anti-pattern",
    message="Avoid 'zypper update' in Dockerfiles. Use specific base image versions for reproducible builds."
)
def avoid_zypper_update():
    return instruction(type="RUN", contains="zypper update")
