from rules.python_decorators import python_rule
from codepathfinder import calls, QueryType

class OSModule(QueryType):
    fqns = ["os"]


@python_rule(
    id="PYTHON-LANG-SEC-101",
    name="Insecure File Permissions",
    severity="MEDIUM",
    category="lang",
    cwe="CWE-732",
    tags="python,file-permissions,chmod,cwe-732",
    message="Overly permissive file permissions detected. Restrict to minimum required permissions.",
    owasp="A01:2021",
)
def detect_insecure_permissions():
    """Detects os.chmod/fchmod/lchmod with overly permissive modes."""
    return OSModule.method("chmod", "fchmod", "lchmod")
