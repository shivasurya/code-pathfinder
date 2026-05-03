from codepathfinder.python_decorators import python_rule
from codepathfinder import calls, QueryType

class XMLRPCModule(QueryType):
    fqns = ["xmlrpc", "xmlrpc.client", "xmlrpc.server"]


@python_rule(
    id="PYTHON-LANG-SEC-092",
    name="Insecure xmlrpc Usage",
    severity="MEDIUM",
    category="lang",
    cwe="CWE-611",
    tags="python,xmlrpc,xxe,CWE-611",
    message="xmlrpc is vulnerable to XXE. Use defusedxml.xmlrpc instead.",
    owasp="A05:2021",
)
def detect_xmlrpc():
    """Detects xmlrpc.client and xmlrpc.server usage."""
    return XMLRPCModule.method("ServerProxy", "SimpleXMLRPCServer", "DocXMLRPCServer")
