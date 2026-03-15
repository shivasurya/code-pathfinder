"""
XML Parsing and Template Injection Security Rules for Python

Rules in this file:
- PYTHON-LANG-SEC-090: Insecure XML Parsing (CWE-611)
- PYTHON-LANG-SEC-091: xml.dom.minidom Usage (CWE-611)
- PYTHON-LANG-SEC-092: Insecure xmlrpc Usage (CWE-611)
- PYTHON-LANG-SEC-093: Mako Template Detected (CWE-96)
- PYTHON-LANG-SEC-094: csv.writer Without defusedcsv (CWE-1236)

Security Impact: MEDIUM
CWE: CWE-611 (Improper Restriction of XML External Entity Reference)
OWASP: A05:2021 - Security Misconfiguration

DESCRIPTION:
Python's standard library XML parsers (xml.etree.ElementTree, xml.dom.minidom,
xml.sax, xml.dom.pulldom) are vulnerable to XML External Entity (XXE) attacks by
default. XXE allows attackers to read local files, perform server-side request
forgery (SSRF), and in some configurations achieve remote code execution through
crafted XML documents. The xmlrpc module is similarly vulnerable. Mako templates
execute embedded Python expressions without sandboxing, enabling server-side
template injection (SSTI). CSV writers that do not sanitize cell values may allow
formula injection in spreadsheet applications.

SECURITY IMPLICATIONS:
An XXE attack uses external entity declarations in XML input to read arbitrary
files from the server filesystem (e.g., /etc/passwd, application config files,
private keys), enumerate internal network services via SSRF, or cause
denial-of-service through recursive entity expansion (billion laughs attack).
Mako templates render expressions like ${...} directly as Python code; if
template content originates from user input, attackers can execute arbitrary
server-side code. CSV formula injection occurs when cell values starting with
=, +, -, or @ are interpreted as formulas by spreadsheet applications, enabling
data exfiltration or command execution on the recipient's machine.

    # Attack scenario: XXE file read
    # Attacker submits XML: <!DOCTYPE foo [<!ENTITY xxe SYSTEM "file:///etc/passwd">]>
    # <root>&xxe;</root>
    tree = ET.parse(attacker_xml)  # Reads /etc/passwd and includes it in the document

VULNERABLE EXAMPLE:
```python
import xml.etree.ElementTree as ET
from xml.dom.minidom import parseString
# XXE-vulnerable XML parsing
tree = ET.parse(user_uploaded_xml)
doc = parseString(user_xml_string)
# Unsandboxed Mako template with user input
from mako.template import Template
t = Template(user_template_string)
output = t.render(data=data)  # SSTI if user controls template
# CSV without formula injection protection
import csv
writer = csv.writer(open("export.csv", "w"))
writer.writerow([user_input])  # May contain =cmd|'/C calc'
```

SECURE EXAMPLE:
```python
# Use defusedxml for safe XML parsing
import defusedxml.ElementTree as ET
tree = ET.parse(user_uploaded_xml)  # XXE protection enabled
import defusedxml.minidom
doc = defusedxml.minidom.parseString(user_xml_string)
# Restrict Mako templates to trusted sources only
from mako.template import Template
t = Template(filename="/app/templates/report.html")  # Trusted file, not user input
# Sanitize CSV cell values to prevent formula injection
def sanitize_csv_value(value):
    if isinstance(value, str) and value and value[0] in "=+-@":
        return "'" + value  # Prefix with single quote
    return value
```

DETECTION AND PREVENTION:
- Replace all xml.etree.ElementTree usage with defusedxml.ElementTree
- Replace xml.dom.minidom with defusedxml.minidom
- Replace xmlrpc with defusedxml.xmlrpc monkey patching
- Never render user-controlled strings as Mako templates
- Sanitize CSV cell values by prefixing dangerous characters with single quotes
- Disable external entity processing if defusedxml is not available

COMPLIANCE:
- CWE-611: Improper Restriction of XML External Entity Reference
- CWE-96: Improper Neutralization of Directives in Statically Saved Code
- CWE-1236: Improper Neutralization of Formula Elements in a CSV File
- OWASP A03:2021 - Injection
- OWASP A05:2021 - Security Misconfiguration
- SANS Top 25 (2023) - CWE-611: XXE

REFERENCES:
- https://cwe.mitre.org/data/definitions/611.html
- https://cwe.mitre.org/data/definitions/96.html
- https://cwe.mitre.org/data/definitions/1236.html
- https://owasp.org/Top10/A05_2021-Security_Misconfiguration/
- https://pypi.org/project/defusedxml/
- https://docs.python.org/3/library/xml.html#xml-vulnerabilities
- https://cheatsheetseries.owasp.org/cheatsheets/XML_External_Entity_Prevention_Cheat_Sheet.html
"""

from rules.python_decorators import python_rule
from codepathfinder import calls, QueryType


class XMLModule(QueryType):
    fqns = ["xml.etree.ElementTree"]


class XMLRPCModule(QueryType):
    fqns = ["xmlrpc", "xmlrpc.client", "xmlrpc.server"]


class MinidomModule(QueryType):
    fqns = ["xml.dom.minidom", "xml.sax", "xml.dom.pulldom"]


class MakoModule(QueryType):
    fqns = ["mako.template"]


class CSVModule(QueryType):
    fqns = ["csv"]


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


@python_rule(
    id="PYTHON-LANG-SEC-091",
    name="xml.dom.minidom Usage",
    severity="MEDIUM",
    category="lang",
    cwe="CWE-611",
    tags="python,xml,xxe,minidom,cwe-611",
    message="xml.dom.minidom is vulnerable to XXE. Use defusedxml.minidom instead.",
    owasp="A05:2021",
)
def detect_minidom():
    """Detects xml.dom.minidom and xml.sax usage."""
    return MinidomModule.method("parse", "parseString")


@python_rule(
    id="PYTHON-LANG-SEC-092",
    name="Insecure xmlrpc Usage",
    severity="MEDIUM",
    category="lang",
    cwe="CWE-611",
    tags="python,xmlrpc,xxe,cwe-611",
    message="xmlrpc is vulnerable to XXE. Use defusedxml.xmlrpc instead.",
    owasp="A05:2021",
)
def detect_xmlrpc():
    """Detects xmlrpc.client and xmlrpc.server usage."""
    return XMLRPCModule.method("ServerProxy", "SimpleXMLRPCServer", "DocXMLRPCServer")


@python_rule(
    id="PYTHON-LANG-SEC-093",
    name="Mako Template Detected",
    severity="MEDIUM",
    category="lang",
    cwe="CWE-96",
    tags="python,mako,template,ssti,cwe-96",
    message="Mako templates do not sandbox expressions. Ensure templates are trusted.",
    owasp="A03:2021",
)
def detect_mako():
    """Detects mako.template.Template usage."""
    return MakoModule.method("Template")


@python_rule(
    id="PYTHON-LANG-SEC-094",
    name="csv.writer Without defusedcsv",
    severity="LOW",
    category="lang",
    cwe="CWE-1236",
    tags="python,csv,csv-injection,defusedcsv,cwe-1236",
    message="csv.writer() detected. Consider defusedcsv to prevent formula injection.",
    owasp="A03:2021",
)
def detect_csv_writer():
    """Detects csv.writer usage."""
    return CSVModule.method("writer", "DictWriter")
