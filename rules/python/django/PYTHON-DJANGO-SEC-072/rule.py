from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class PickleModule(QueryType):
    fqns = ["pickle", "_pickle", "cPickle"]

class YamlModule(QueryType):
    fqns = ["yaml"]

class DillModule(QueryType):
    fqns = ["dill"]

_DJANGO_SOURCES = [
    calls("request.GET.get"),
    calls("request.POST.get"),
    calls("request.GET"),
    calls("request.POST"),
    calls("request.COOKIES.get"),
    calls("request.FILES.get"),
    calls("*.GET.get"),
    calls("*.POST.get"),
]


@python_rule(
    id="PYTHON-DJANGO-SEC-072",
    name="Django Insecure Deserialization of Request Data",
    severity="CRITICAL",
    category="django",
    cwe="CWE-502",
    tags="python,django,deserialization,pickle,yaml,OWASP-A08,CWE-502",
    message="Request data flows to unsafe deserialization. Use JSON instead of pickle/yaml.",
    owasp="A08:2021",
)
def detect_django_insecure_deserialization():
    """Detects request data flowing to pickle/yaml/dill/shelve deserialization."""
    return flows(
        from_sources=_DJANGO_SOURCES,
        to_sinks=[
            PickleModule.method("loads", "load"),
            YamlModule.method("load", "unsafe_load"),
            DillModule.method("loads", "load"),
            calls("shelve.open"),
        ],
        sanitized_by=[],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
