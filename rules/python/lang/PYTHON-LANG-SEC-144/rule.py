from rules.python_decorators import python_rule
from codepathfinder import calls, QueryType


@python_rule(
    id="PYTHON-LANG-SEC-144",
    name="Insecure CORS Wildcard Configuration",
    severity="HIGH",
    category="lang",
    cwe="CWE-942",
    tags="python,cors,misconfiguration,wildcard,OWASP-A05,CWE-942",
    message="CORS middleware with permissive configuration detected. Review allow_origins and allow_credentials settings.",
    owasp="A05:2021",
)
def detect_cors_wildcard():
    """Detects CORSMiddleware and CORS() calls that may use wildcard origins."""
    return calls("CORSMiddleware", "*.add_middleware", "CORS")
