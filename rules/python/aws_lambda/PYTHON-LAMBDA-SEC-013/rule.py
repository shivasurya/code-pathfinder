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
    id="PYTHON-LAMBDA-SEC-013",
    name="Lambda SQL Injection via PyMySQL Cursor",
    severity="CRITICAL",
    category="aws_lambda",
    cwe="CWE-89",
    tags="python,aws,lambda,sql-injection,pymysql,owasp-a03,cwe-89",
    message="Lambda event data flows to PyMySQL cursor.execute(). Use parameterized queries.",
    owasp="A03:2021",
)
def detect_lambda_pymysql_sqli():
    """Detects Lambda event data flowing to PyMySQL cursor."""
    return flows(
        from_sources=_LAMBDA_SOURCES,
        to_sinks=[
            DBCursor.method("execute", "executemany").tracks(0),
            calls("cursor.execute"),
        ],
        sanitized_by=[
            calls("escape"),
            calls("escape_string"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
