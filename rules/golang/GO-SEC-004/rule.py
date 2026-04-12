"""GO-SEC-004: Hardcoded credentials (passwords, API keys, tokens) in source code."""

from codepathfinder import variable, Or
from rules.go_decorators import go_rule


@go_rule(
    id="GO-SEC-004",
    severity="HIGH",
    cwe="CWE-798",
    owasp="A07:2021",
    tags="go,security,hardcoded-credentials,secrets,CWE-798,OWASP-A07",
    message=(
        "Detected a variable with a name suggesting credential storage "
        "(password, secret, api_key, token, etc.) being passed to a function. "
        "Hardcoded credentials in source code can be extracted from repositories, "
        "compiled binaries, or version control history. "
        "Use environment variables (os.Getenv) or a secrets manager instead."
    ),
)
def go_hardcoded_credentials():
    """Detects hardcoded passwords, API keys, and tokens passed to functions in Go."""
    return Or(
        variable(pattern="*password*"),
        variable(pattern="*secret*"),
        variable(pattern="*api_key*"),
        variable(pattern="*apikey*"),
        variable(pattern="*token*"),
        variable(pattern="*credential*"),
    )
