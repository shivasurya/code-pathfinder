from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class AsyncioModule(QueryType):
    fqns = ["asyncio"]


@python_rule(
    id="PYTHON-LANG-SEC-022",
    name="Dangerous asyncio Shell Execution",
    severity="HIGH",
    category="lang",
    cwe="CWE-78",
    tags="python,asyncio,shell,command-injection,CWE-78",
    message="asyncio.create_subprocess_shell() detected. Use create_subprocess_exec() instead.",
    owasp="A03:2021",
)
def detect_asyncio_shell():
    """Detects asyncio.create_subprocess_shell and related methods."""
    return AsyncioModule.method("create_subprocess_shell")
