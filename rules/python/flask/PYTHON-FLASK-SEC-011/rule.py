from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class RequestsLib(QueryType):
    fqns = ["requests"]


@python_rule(
    id="PYTHON-FLASK-SEC-011",
    name="Flask Tainted URL Host",
    severity="HIGH",
    category="flask",
    cwe="CWE-918",
    tags="python,flask,ssrf,url-host,OWASP-A10,CWE-918",
    message="User input used in URL host construction. Validate against an allowlist of hosts.",
    owasp="A10:2021",
)
def detect_flask_tainted_url_host():
    """Detects Flask request data used in URL host construction flowing to HTTP requests."""
    return flows(
        from_sources=[
            calls("request.args.get"),
            calls("request.form.get"),
            calls("request.values.get"),
            calls("request.get_json"),
        ],
        to_sinks=[
            RequestsLib.method("get", "post", "put", "delete").tracks(0),
            calls("http_requests.get"),
            calls("http_requests.post"),
            calls("urllib.request.urlopen"),
        ],
        sanitized_by=[
            calls("*.validate_host"),
            calls("*.is_safe_url"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
