from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class Builtins(QueryType):
    fqns = ["builtins"]

_LAMBDA_SOURCES = [
    calls("event.get"),
    calls("event.items"),
    calls("event.values"),
    calls("*.get"),
]


@python_rule(
    id="PYTHON-LAMBDA-SEC-022",
    name="Lambda Code Injection via eval/exec",
    severity="CRITICAL",
    category="aws_lambda",
    cwe="CWE-95",
    tags="python,aws,lambda,code-injection,eval,exec,OWASP-A03,CWE-95",
    message="Lambda event data flows to eval()/exec()/compile(). Never eval untrusted data.",
    owasp="A03:2021",
)
def detect_lambda_code_injection():
    """Detects Lambda event data flowing to eval/exec/compile."""
    return flows(
        from_sources=_LAMBDA_SOURCES,
        to_sinks=[
            Builtins.method("eval", "exec", "compile"),
            calls("eval"),
            calls("exec"),
            calls("compile"),
        ],
        sanitized_by=[
            calls("ast.literal_eval"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
