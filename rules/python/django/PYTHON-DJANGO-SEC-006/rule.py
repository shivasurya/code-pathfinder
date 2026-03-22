from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

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
    id="PYTHON-DJANGO-SEC-006",
    name="Tainted SQL String Construction",
    severity="HIGH",
    category="django",
    cwe="CWE-89",
    tags="python,django,sql-injection,string-format,OWASP-A03,CWE-89",
    message="User input used in SQL string construction. Use parameterized queries.",
    owasp="A03:2021",
)
def detect_django_tainted_sql_string():
    """Detects request data used in string formatting that reaches execute()."""
    return flows(
        from_sources=_DJANGO_SOURCES,
        to_sinks=[
            calls("cursor.execute"),
            calls("*.execute"),
        ],
        sanitized_by=[
            calls("escape"),
            calls("escape_string"),
            calls("escape_sql"),
            calls("int"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
