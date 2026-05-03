from codepathfinder.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class DjangoORM(QueryType):
    fqns = ["django.db.models.Manager", "django.db.models.QuerySet"]
    patterns = ["*Manager", "*QuerySet"]
    match_subclasses = True

# Common Django request sources
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
    id="PYTHON-DJANGO-SEC-002",
    name="Django SQL Injection via ORM .raw()",
    severity="CRITICAL",
    category="django",
    cwe="CWE-89",
    tags="python,django,sql-injection,orm-raw,OWASP-A03,CWE-89",
    message="User input flows to .raw() query. Use parameterized .raw() with %s placeholders.",
    owasp="A03:2021",
)
def detect_django_raw_sqli():
    """Detects request data flowing to Model.objects.raw()."""
    return flows(
        from_sources=_DJANGO_SOURCES,
        to_sinks=[
            DjangoORM.method("raw").tracks(0),
            calls("*.objects.raw"),
            calls("*.raw"),
        ],
        sanitized_by=[
            calls("escape"),
            calls("escape_string"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
