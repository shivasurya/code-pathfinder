from rules.python_decorators import python_rule
from codepathfinder import calls, QueryType


@python_rule(
    id="PYTHON-LANG-SEC-136",
    name="Dynamic Module Import from User Input",
    severity="HIGH",
    category="lang",
    cwe="CWE-470",
    tags="python,import,dynamic-import,code-injection,CWE-470,OWASP-A03",
    message="Dynamic module import detected (importlib.import_module, load_object, __import__). Importing modules based on user input can lead to arbitrary code execution.",
    owasp="A03:2021",
)
def detect_dynamic_import():
    """Detects dynamic module imports that may use user-controlled input."""
    return calls("importlib.import_module", "*.import_module", "load_object", "*.load_object", "__import__")
