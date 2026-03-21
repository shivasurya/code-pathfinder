from rules.python_decorators import python_rule
from codepathfinder import calls, QueryType

class RequestsLib(QueryType):
    fqns = ["requests"]


@python_rule(
    id="PYTHON-LANG-SEC-053",
    name="Disabled Certificate Validation",
    severity="HIGH",
    category="lang",
    cwe="CWE-295",
    tags="python,ssl,cert-validation,mitm,cwe-295",
    message="Certificate validation disabled (verify=False or CERT_NONE). Enable certificate verification.",
    owasp="A07:2021",
)
def detect_disabled_cert():
    """Detects requests.get(verify=False) and similar patterns."""
    return RequestsLib.method("get", "post", "put", "delete",
                              "patch", "head", "request").where("verify", False)
