from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

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
    id="PYTHON-PYRAMID-SEC-001",
    name="Pyramid CSRF Check Disabled Globally",
    severity="HIGH",
    category="pyramid",
    cwe="CWE-352",
    tags="python,pyramid,csrf,security,OWASP-A05,CWE-352",
    message="CSRF protection disabled globally via set_default_csrf_options(require_csrf=False).",
    owasp="A05:2021",
)
def detect_pyramid_csrf_disabled():
    """Detects Configurator.set_default_csrf_options() calls."""
    return calls("*.set_default_csrf_options")
