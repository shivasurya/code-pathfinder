from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class CodeModule(QueryType):
    fqns = ["code"]


@python_rule(
    id="PYTHON-LANG-SEC-003",
    name="Dangerous code.InteractiveConsole Usage",
    severity="HIGH",
    category="lang",
    cwe="CWE-95",
    tags="python,code-run,interactive-console,cwe-95",
    message="code.InteractiveConsole/interact detected. This enables arbitrary code execution.",
    owasp="A03:2021",
)
def detect_code_run():
    """Detects code.InteractiveConsole and code.interact usage."""
    return CodeModule.method("InteractiveConsole", "interact", "compile_command")
