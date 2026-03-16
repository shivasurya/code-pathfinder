from rules.python_decorators import python_rule
from codepathfinder import calls, QueryType

class XMLModule(QueryType):
    fqns = ["xml.etree.ElementTree"]


@python_rule(
    id="PYTHON-LANG-SEC-090",
    name="Insecure XML Parsing",
    severity="MEDIUM",
    category="lang",
    cwe="CWE-611",
    tags="python,xml,xxe,defusedxml,owasp-a05,cwe-611",
    message="xml.etree.ElementTree is vulnerable to XXE. Use defusedxml instead.",
    owasp="A05:2021",
)
def detect_insecure_xml():
    """Detects xml.etree.ElementTree.parse/fromstring usage."""
    return XMLModule.method("parse", "fromstring", "iterparse", "XMLParser")
