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
    id="PYTHON-DJANGO-SEC-005",
    name="Raw SQL Usage Detected (Audit)",
    severity="MEDIUM",
    category="django",
    cwe="CWE-89",
    tags="python,django,sql-injection,raw-sql,audit,cwe-89",
    message="Raw SQL usage detected. Ensure parameterized queries are used.",
    owasp="A03:2021",
)
def detect_django_raw_sql_audit():
    """Audit rule: detects any usage of raw SQL APIs."""
    return DjangoExpressions.method("RawSQL")
