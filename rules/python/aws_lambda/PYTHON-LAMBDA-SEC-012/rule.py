from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class DBCursor(QueryType):
    fqns = ["sqlite3.Cursor", "mysql.connector.cursor.MySQLCursor",
            "psycopg2.extensions.cursor", "pymysql.cursors.Cursor"]
    patterns = ["*Cursor"]
    match_subclasses = True

_LAMBDA_SOURCES = [
    calls("event.get"),
    calls("event.items"),
    calls("event.values"),
    calls("*.get"),
]


@python_rule(
    id="PYTHON-LAMBDA-SEC-012",
    name="Lambda SQL Injection via pymssql Cursor",
    severity="CRITICAL",
    category="aws_lambda",
    cwe="CWE-89",
    tags="python,aws,lambda,sql-injection,pymssql,OWASP-A03,CWE-89",
    message="Lambda event data flows to pymssql cursor.execute(). Use parameterized queries.",
    owasp="A03:2021",
)
def detect_lambda_pymssql_sqli():
    """Detects Lambda event data flowing to pymssql cursor."""
    return flows(
        from_sources=_LAMBDA_SOURCES,
        to_sinks=[
            DBCursor.method("execute").tracks(0),
            calls("cursor.execute"),
        ],
        sanitized_by=[
            calls("escape"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
