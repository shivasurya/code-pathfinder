from rules.python_decorators import python_rule
from codepathfinder import calls, QueryType

class TarFileModule(QueryType):
    fqns = ["tarfile"]


@python_rule(
    id="PYTHON-LANG-SEC-110",
    name="Unsafe Tarfile Extraction Detected",
    severity="HIGH",
    category="lang",
    cwe="CWE-22",
    tags="python,tarfile,path-traversal,zip-slip,CWE-22,OWASP-A01",
    message="tarfile.extractall() or tarfile.extract() detected. Tar archives can contain path traversal entries. Use filter='data' or validate members before extraction.",
    owasp="A01:2021",
)
def detect_tarfile_extract():
    """Detects tarfile.extractall() and tarfile.extract() usage."""
    return TarFileModule.method("extractall", "extract")
