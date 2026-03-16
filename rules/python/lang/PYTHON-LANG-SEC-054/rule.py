from rules.python_decorators import python_rule
from codepathfinder import calls, QueryType

class HttplibModule(QueryType):
    fqns = ["http.client"]


@python_rule(
    id="PYTHON-LANG-SEC-054",
    name="Insecure HTTP Connection",
    severity="MEDIUM",
    category="lang",
    cwe="CWE-319",
    tags="python,http,plaintext,insecure-transport,cwe-319",
    message="HTTPConnection used instead of HTTPSConnection. Use HTTPS for sensitive communications.",
    owasp="A02:2021",
)
def detect_http_connection():
    """Detects http.client.HTTPConnection usage."""
    return HttplibModule.method("HTTPConnection")
