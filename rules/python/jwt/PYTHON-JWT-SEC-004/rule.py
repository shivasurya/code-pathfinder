from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class JWTModule(QueryType):
    fqns = ["jwt"]


@python_rule(
    id="PYTHON-JWT-SEC-004",
    name="JWT Exposed Credentials (Audit)",
    severity="MEDIUM",
    category="jwt",
    cwe="CWE-522",
    tags="python,jwt,credentials,audit,CWE-522",
    message="jwt.encode() detected. Ensure no passwords or secrets are in the payload.",
    owasp="A02:2021",
)
def detect_jwt_exposed_credentials():
    """Audit: detects jwt.encode() to review for credential exposure."""
    return JWTModule.method("encode")
