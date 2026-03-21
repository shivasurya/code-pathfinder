from rules.python_decorators import python_rule
from codepathfinder import calls, QueryType

class UrllibModule(QueryType):
    fqns = ["urllib.request"]


@python_rule(
    id="PYTHON-LANG-SEC-061",
    name="Insecure urllib.urlopen",
    severity="MEDIUM",
    category="lang",
    cwe="CWE-319",
    tags="python,urllib,http,insecure-transport,cwe-319",
    message="urllib.request.urlopen() detected. Ensure HTTPS URLs are used.",
    owasp="A02:2021",
)
def detect_urllib_insecure():
    """Detects urllib.request.urlopen and urlretrieve calls."""
    return UrllibModule.method("urlopen", "urlretrieve")
