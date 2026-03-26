from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class ImportlibModule(QueryType):
    fqns = ["importlib"]


@python_rule(
    id="PYTHON-LANG-SEC-136",
    name="Dynamic Module Import from User Input",
    severity="HIGH",
    category="lang",
    cwe="CWE-470",
    tags="python,import,dynamic-import,code-injection,CWE-470,OWASP-A03",
    message="User input flows to dynamic module import. Importing modules based on user input can lead to arbitrary code execution.",
    owasp="A03:2021",
)
def detect_dynamic_import():
    """Detects user input flowing to dynamic module import functions."""
    return flows(
        from_sources=[
            calls("request.headers.get"),
            calls("request.form.get"),
            calls("request.args.get"),
            calls("request.GET.get"),
            calls("request.POST.get"),
            calls("*.get"),
        ],
        to_sinks=[
            ImportlibModule.method("import_module"),
            calls("load_object"),
            calls("*.load_object"),
            calls("__import__"),
        ],
        sanitized_by=[],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
