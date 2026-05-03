from codepathfinder.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class SubprocessModule(QueryType):
    fqns = ["subprocess"]


@python_rule(
    id="PYTHON-FLASK-SEC-002",
    name="Flask Command Injection via subprocess",
    severity="CRITICAL",
    category="flask",
    cwe="CWE-78",
    tags="python,flask,command-injection,subprocess,OWASP-A03,CWE-78",
    message="User input flows to subprocess call. Use shlex.quote() or avoid shell=True.",
    owasp="A03:2021",
)
def detect_flask_subprocess_injection():
    """Detects Flask request data flowing to subprocess functions."""
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
            SubprocessModule.method("call", "check_call", "check_output",
                                    "run", "Popen", "getoutput", "getstatusoutput").tracks(0),
        ],
        sanitized_by=[
            calls("shlex.quote"),
            calls("shlex.split"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
