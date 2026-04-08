from rules.python_decorators import python_rule
from codepathfinder import calls, QueryType


class ZipFileModule(QueryType):
    fqns = ["zipfile"]


@python_rule(
    id="PYTHON-LANG-SEC-157",
    name="ZipFile Extract Path Traversal (Zip Slip)",
    severity="HIGH",
    category="lang",
    cwe="CWE-22",
    tags="python,zipfile,path-traversal,zip-slip,OWASP-A01,CWE-22",
    message="ZipFile.extract() or ZipFile.extractall() detected. Malicious zip entries with '../' paths can write files outside the target directory.",
    owasp="A01:2021",
)
def detect_zipfile_extract():
    """Detects zipfile.ZipFile.extract() and extractall() calls."""
    return ZipFileModule.method("extract", "extractall")
