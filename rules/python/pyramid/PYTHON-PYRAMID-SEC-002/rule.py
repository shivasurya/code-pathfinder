from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class PyramidResponse(QueryType):
    fqns = ["pyramid.response.Response", "pyramid.request.Response"]

_PYRAMID_SOURCES = [
    calls("request.params.get"),
    calls("request.params"),
    calls("request.GET.get"),
    calls("request.POST.get"),
    calls("request.matchdict.get"),
    calls("request.json_body.get"),
    calls("*.params.get"),
    calls("*.params"),
]


@python_rule(
    id="PYTHON-PYRAMID-SEC-002",
    name="Pyramid Direct Response XSS",
    severity="HIGH",
    category="pyramid",
    cwe="CWE-79",
    tags="python,pyramid,xss,response,owasp-a03,cwe-79",
    message="User input flows directly to Response(). Use templates with auto-escaping.",
    owasp="A03:2021",
)
def detect_pyramid_response_xss():
    """Detects request data flowing to Pyramid Response."""
    return flows(
        from_sources=_PYRAMID_SOURCES,
        to_sinks=[
            PyramidResponse.method("__init__"),
            calls("Response"),
            calls("pyramid.response.Response"),
        ],
        sanitized_by=[
            calls("escape"),
            calls("markupsafe.escape"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="local",
    )
