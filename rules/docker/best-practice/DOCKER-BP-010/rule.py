from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction
from rules.container_combinators import all_of


@dockerfile_rule(
    id="DOCKER-BP-010",
    name="Missing pipefail in Shell Commands",
    severity="MEDIUM",
    cwe="CWE-703",
    category="best-practice",
    tags="docker,dockerfile,shell,bash,pipefail,error-handling,best-practice,reliability,build,pipes",
    message="RUN instruction uses pipes without 'set -o pipefail'. This masks failures in piped commands."
)
def set_pipefail():
    return all_of(
        instruction(type="RUN", contains="|"),
        instruction(type="RUN", not_contains="set -o pipefail")
    )
