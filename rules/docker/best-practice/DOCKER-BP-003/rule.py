from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction


@dockerfile_rule(
    id="DOCKER-BP-003",
    name="Deprecated MAINTAINER Instruction",
    severity="INFO",
    cwe="CWE-710",
    category="best-practice",
    tags="docker,dockerfile,maintainer,label,deprecated,metadata,best-practice,oci,standards,legacy",
    message="MAINTAINER instruction is deprecated. Use LABEL org.opencontainers.image.authors instead."
)
def maintainer_deprecated():
    """
    Detects usage of deprecated MAINTAINER instruction.

    The MAINTAINER instruction is deprecated since Docker 1.13 in favor
    of LABEL instructions with standardized OCI metadata keys.
    """
    return instruction(type="MAINTAINER")
