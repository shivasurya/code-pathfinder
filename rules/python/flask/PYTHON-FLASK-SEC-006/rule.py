from codepathfinder.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class RequestsLib(QueryType):
    fqns = ["requests"]

class UrllibRequest(QueryType):
    fqns = ["urllib.request"]


@python_rule(
    id="PYTHON-FLASK-SEC-006",
    name="Flask SSRF via requests library",
    severity="HIGH",
    category="flask",
    cwe="CWE-918",
    tags="python,flask,ssrf,requests,OWASP-A10,CWE-918",
    message="User input flows to HTTP request URL. Validate and allowlist target URLs.",
    owasp="A10:2021",
)
def detect_flask_ssrf():
    """Detects Flask request data flowing to requests library calls."""
    return flows(
        from_sources=[
            calls("request.args.get"),
            calls("request.form.get"),
            calls("request.values.get"),
            calls("request.get_json"),
        ],
        to_sinks=[
            RequestsLib.method("get", "post", "put", "delete", "patch",
                               "head", "options", "request").tracks(0),
            UrllibRequest.method("urlopen", "Request").tracks(0),
            calls("http_requests.get"),
            calls("http_requests.post"),
        ],
        sanitized_by=[
            calls("*.validate_url"),
            calls("*.is_safe_url"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
