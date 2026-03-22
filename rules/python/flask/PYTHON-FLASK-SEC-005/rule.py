from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class Builtins(QueryType):
    fqns = ["builtins"]


@python_rule(
    id="PYTHON-FLASK-SEC-005",
    name="Flask Code Injection via exec()",
    severity="CRITICAL",
    category="flask",
    cwe="CWE-95",
    tags="python,flask,code-injection,exec,rce,OWASP-A03,CWE-95",
    message="User input flows to exec(). Never execute user-supplied code.",
    owasp="A03:2021",
)
def detect_flask_exec_injection():
    """Detects Flask request data flowing to exec()."""
    return flows(
        from_sources=[
            calls("request.args.get"),
            calls("request.form.get"),
            calls("request.values.get"),
            calls("request.get_json"),
        ],
        to_sinks=[
            Builtins.method("exec", "compile").tracks(0),
            calls("exec"),
            calls("compile"),
        ],
        sanitized_by=[],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
