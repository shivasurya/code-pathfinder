from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class DjangoExpressions(QueryType):
    fqns = ["django.db.models.expressions"]

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
    id="PYTHON-DJANGO-SEC-004",
    name="Django SQL Injection via RawSQL Expression",
    severity="CRITICAL",
    category="django",
    cwe="CWE-89",
    tags="python,django,sql-injection,rawsql,owasp-a03,cwe-89",
    message="User input flows to RawSQL() expression. Use parameterized queries.",
    owasp="A03:2021",
)
def detect_django_rawsql_sqli():
    """Detects request data flowing to RawSQL()."""
    return flows(
        from_sources=_DJANGO_SOURCES,
        to_sinks=[
            DjangoExpressions.method("RawSQL").tracks(0),
            calls("RawSQL"),
            calls("*.RawSQL"),
        ],
        sanitized_by=[
            calls("escape"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
