from codepathfinder.python_decorators import python_rule
from codepathfinder import calls, QueryType

class UrllibModule(QueryType):
    fqns = ["urllib.request"]


@python_rule(
    id="PYTHON-LANG-SEC-062",
    name="Insecure urllib Request Object",
    severity="MEDIUM",
    category="lang",
    cwe="CWE-319",
    tags="python,urllib,request-object,insecure-transport,CWE-319",
    message="urllib.request.Request() detected. Ensure HTTPS URLs are used.",
    owasp="A02:2021",
)
def detect_urllib_request():
    """Detects urllib.request.Request and OpenerDirector usage."""
    return UrllibModule.method("Request", "OpenerDirector")
