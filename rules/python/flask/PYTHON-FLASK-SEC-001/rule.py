from codepathfinder.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class OSModule(QueryType):
    fqns = ["os"]


@python_rule(
    id="PYTHON-FLASK-SEC-001",
    name="Flask Command Injection via os.system",
    severity="CRITICAL",
    category="flask",
    cwe="CWE-78",
    tags="python,flask,command-injection,os-system,OWASP-A03,CWE-78",
    message="User input flows to os.system(). Use subprocess with list args and shlex.quote() instead.",
    owasp="A03:2021",
)
def detect_flask_os_system_injection():
    """Detects Flask request data flowing to os.system() or os.popen()."""
    return flows(
        from_sources=[
            calls("request.args.get"),
            calls("request.form.get"),
            calls("request.values.get"),
            calls("request.get_json"),
            calls("request.cookies.get"),
            calls("request.headers.get"),
        ],
        to_sinks=[
            OSModule.method("system", "popen", "popen2", "popen3", "popen4").tracks(0),
        ],
        sanitized_by=[
            calls("shlex.quote"),
            calls("shlex.split"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
