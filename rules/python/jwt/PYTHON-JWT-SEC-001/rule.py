from codepathfinder.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class JWTModule(QueryType):
    fqns = ["jwt"]


@python_rule(
    id="PYTHON-JWT-SEC-001",
    name="JWT Hardcoded Secret",
    severity="HIGH",
    category="jwt",
    cwe="CWE-522",
    tags="python,jwt,hardcoded-secret,credentials,OWASP-A02,CWE-522",
    message="Hardcoded string used as JWT signing secret. Use environment variables or key management.",
    owasp="A02:2021",
)
def detect_jwt_hardcoded_secret():
    """Detects jwt.encode() with hardcoded string secret."""
    return JWTModule.method("encode")
