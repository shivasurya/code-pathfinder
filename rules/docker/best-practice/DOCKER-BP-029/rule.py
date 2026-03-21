from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction


@dockerfile_rule(
    id="DOCKER-BP-029",
    name="Avoid yum update",
    severity="MEDIUM",
    cwe="CWE-710",
    category="best-practice",
    tags="docker,dockerfile,yum,package-manager,centos,rhel,update,reproducibility,best-practice,anti-pattern",
    message="Avoid 'yum update' in Dockerfiles. Use specific base image versions for reproducible builds."
)
def avoid_yum_update():
    return instruction(type="RUN", contains="yum update")
