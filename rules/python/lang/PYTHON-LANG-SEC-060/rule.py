from codepathfinder.python_decorators import python_rule
from codepathfinder import calls, QueryType

class RequestsLib(QueryType):
    fqns = ["requests"]


@python_rule(
    id="PYTHON-LANG-SEC-060",
    name="HTTP Request Without TLS",
    severity="MEDIUM",
    category="lang",
    cwe="CWE-319",
    tags="python,requests,http,insecure-transport,CWE-319",
    message="HTTP URL used in requests call. Use HTTPS for sensitive data transmission.",
    owasp="A02:2021",
)
def detect_requests_http():
    """Detects requests library calls (audit for HTTP URLs)."""
    return RequestsLib.method("get", "post", "put", "delete", "patch", "head", "request")
