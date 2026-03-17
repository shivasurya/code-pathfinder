from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class JWTModule(QueryType):
    fqns = ["jwt"]


@python_rule(
    id="PYTHON-JWT-SEC-002",
    name="JWT None Algorithm",
    severity="CRITICAL",
    category="jwt",
    cwe="CWE-327",
    tags="python,jwt,none-algorithm,weak-crypto,owasp-a02,cwe-327",
    message="JWT with algorithm='none' disables signature verification. Use HS256 or RS256.",
    owasp="A02:2021",
)
def detect_jwt_none_algorithm():
    """Detects jwt.encode() with algorithm='none'."""
    return JWTModule.method("encode").where("algorithm", "none")
