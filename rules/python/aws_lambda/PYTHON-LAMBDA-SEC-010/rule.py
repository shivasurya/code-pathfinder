from codepathfinder.python_decorators import python_rule
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
    id="PYTHON-LAMBDA-SEC-010",
    name="Lambda SQL Injection via MySQL Cursor",
    severity="CRITICAL",
    category="aws_lambda",
    cwe="CWE-89",
    tags="python,aws,lambda,sql-injection,mysql,OWASP-A03,CWE-89",
    message="Lambda event data flows to MySQL cursor.execute(). Use parameterized queries.",
    owasp="A03:2021",
)
def detect_lambda_mysql_sqli():
    """Detects Lambda event data flowing to MySQL cursor.execute()."""
    return flows(
        from_sources=_LAMBDA_SOURCES,
        to_sinks=[
            DBCursor.method("execute", "executemany").tracks(0),
        ],
        sanitized_by=[
            calls("escape"),
            calls("escape_string"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
