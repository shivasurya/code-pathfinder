from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class JWTModule(QueryType):
    fqns = ["jwt"]


@python_rule(
    id="PYTHON-JWT-SEC-003",
    name="Unverified JWT Decode",
    severity="HIGH",
    category="jwt",
    cwe="CWE-287",
    tags="python,jwt,unverified,authentication,OWASP-A07,CWE-287",
    message="jwt.decode() with verify_signature=False. Token integrity is not checked.",
    owasp="A07:2021",
)
def detect_jwt_unverified_decode():
    """Detects jwt.decode() calls that may disable verification."""
    return JWTModule.method("decode")
