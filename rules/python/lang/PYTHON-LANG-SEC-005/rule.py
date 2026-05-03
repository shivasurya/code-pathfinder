from codepathfinder.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets


@python_rule(
    id="PYTHON-LANG-SEC-005",
    name="Non-literal import Detected",
    severity="MEDIUM",
    category="lang",
    cwe="CWE-95",
    tags="python,import,dynamic-import,CWE-95",
    message="__import__() or importlib.import_module() with non-literal argument detected.",
    owasp="A03:2021",
)
def detect_non_literal_import():
    """Detects dynamic imports via __import__ and importlib."""
    return calls("__import__", "importlib.import_module")
