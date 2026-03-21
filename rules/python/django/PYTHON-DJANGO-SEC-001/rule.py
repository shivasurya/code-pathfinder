from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class DBCursor(QueryType):
    fqns = ["sqlite3.Cursor", "mysql.connector.cursor.MySQLCursor",
            "psycopg2.extensions.cursor", "pymysql.cursors.Cursor"]
    patterns = ["*Cursor"]
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
    id="PYTHON-DJANGO-SEC-001",
    name="Django SQL Injection via cursor.execute()",
    severity="CRITICAL",
    category="django",
    cwe="CWE-89",
    tags="python,django,sql-injection,cursor,owasp-a03,cwe-89",
    message="User input flows to cursor.execute() without parameterization. Use %s placeholders.",
    owasp="A03:2021",
)
def detect_django_cursor_sqli():
    """Detects request data flowing to cursor.execute()."""
    return flows(
        from_sources=_DJANGO_SOURCES,
        to_sinks=[
            DBCursor.method("execute", "executemany").tracks(0),
            calls("cursor.execute"),
            calls("*.execute"),
        ],
        sanitized_by=[
            calls("escape"),
            calls("escape_string"),
            calls("escape_sql"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
