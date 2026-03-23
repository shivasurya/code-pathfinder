from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class Builtins(QueryType):
    fqns = ["builtins"]


@python_rule(
    id="PYTHON-FLASK-SEC-004",
    name="Flask Code Injection via eval()",
    severity="CRITICAL",
    category="flask",
    cwe="CWE-95",
    tags="python,flask,code-injection,eval,rce,OWASP-A03,CWE-95",
    message="User input flows to eval(). Use ast.literal_eval() for safe evaluation.",
    owasp="A03:2021",
)
def detect_flask_eval_injection():
    """Detects Flask request data flowing to eval()."""
    return flows(
        from_sources=[
            calls("request.args.get"),
            calls("request.form.get"),
            calls("request.values.get"),
            calls("request.get_json"),
        ],
        to_sinks=[
            Builtins.method("eval").tracks(0),
            calls("eval"),
        ],
        sanitized_by=[
            calls("ast.literal_eval"),
            calls("json.loads"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
