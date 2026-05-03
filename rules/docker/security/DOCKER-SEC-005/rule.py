from codepathfinder.container_decorators import dockerfile_rule
from codepathfinder.container_matchers import instruction


@dockerfile_rule(
    id="DOCKER-SEC-005",
    name="Secret in Build Argument",
    severity="CRITICAL",
    cwe="CWE-538",
    category="security",
    tags="docker,dockerfile,secrets,credentials,security,arg,build-arg,password,token,api-key,sensitive-data,information-disclosure",
    message="Build argument name suggests it contains a secret. ARG values are visible in image history via 'docker history'."
)
def secret_in_build_arg():
    """
    Detects ARG instructions with names suggesting secrets.

    Build arguments are stored in the image layer history and can be
    retrieved by anyone with access to the image. Never pass secrets
    as build arguments.
    """
    return instruction(
        type="ARG",
        arg_name_regex=r"(?i)^.*(password|passwd|secret|token|key|apikey|api_key|auth|credential|cred|private|access_token|client_secret).*$"
    )
