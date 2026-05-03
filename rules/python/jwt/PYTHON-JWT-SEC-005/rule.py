from codepathfinder.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class JWTModule(QueryType):
    fqns = ["jwt"]


@python_rule(
    id="PYTHON-JWT-SEC-005",
    name="JWT Exposed Data (Audit)",
    severity="LOW",
    category="jwt",
    cwe="CWE-522",
    tags="python,jwt,data-exposure,audit,CWE-522",
    message="Data passed to jwt.encode(). JWT payloads are base64-encoded, not encrypted.",
    owasp="A02:2021",
)
def detect_jwt_exposed_data():
    """Audit: detects any data flowing into jwt.encode()."""
    return flows(
        from_sources=[
            calls("request.GET.get"),
            calls("request.POST.get"),
            calls("request.args.get"),
            calls("request.form.get"),
        ],
        to_sinks=[
            JWTModule.method("encode"),
        ],
        sanitized_by=[],
        propagates_through=PropagationPresets.standard(),
        scope="local",
    )
