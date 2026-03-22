from rules.python_decorators import python_rule
from codepathfinder import calls, QueryType

class MinidomModule(QueryType):
    fqns = ["xml.dom.minidom", "xml.sax", "xml.dom.pulldom"]


@python_rule(
    id="PYTHON-LANG-SEC-091",
    name="xml.dom.minidom Usage",
    severity="MEDIUM",
    category="lang",
    cwe="CWE-611",
    tags="python,xml,xxe,minidom,CWE-611",
    message="xml.dom.minidom is vulnerable to XXE. Use defusedxml.minidom instead.",
    owasp="A05:2021",
)
def detect_minidom():
    """Detects xml.dom.minidom and xml.sax usage."""
    return MinidomModule.method("parse", "parseString")
