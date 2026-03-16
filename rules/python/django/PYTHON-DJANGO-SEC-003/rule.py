from rules.python_decorators import python_rule
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
    id="PYTHON-DJANGO-SEC-003",
    name="Django SQL Injection via ORM .extra()",
    severity="HIGH",
    category="django",
    cwe="CWE-89",
    tags="python,django,sql-injection,orm-extra,owasp-a03,cwe-89",
    message="User input flows to .extra() query. Use .annotate() or parameterized queries instead.",
    owasp="A03:2021",
)
def detect_django_extra_sqli():
    """Detects request data flowing to QuerySet.extra()."""
    return flows(
        from_sources=_DJANGO_SOURCES,
        to_sinks=[
            DjangoORM.method("extra").tracks(0),
            calls("*.objects.extra"),
            calls("*.extra"),
        ],
        sanitized_by=[
            calls("escape"),
            calls("escape_string"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
