from rules.python_decorators import python_rule
from codepathfinder import calls, QueryType

class MakoModule(QueryType):
    fqns = ["mako.template"]


@python_rule(
    id="PYTHON-LANG-SEC-093",
    name="Mako Template Detected",
    severity="MEDIUM",
    category="lang",
    cwe="CWE-94",
    tags="python,mako,template,ssti,cwe-94",
    message="Mako templates do not sandbox expressions. Ensure templates are trusted.",
    owasp="A03:2021",
)
def detect_mako():
    """Detects mako.template.Template usage."""
    return MakoModule.method("Template")
