from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets


class FlaskRequest(QueryType):
    fqns = ["flask"]
    patterns = ["*request"]


class DBCursor(QueryType):
    fqns = ["sqlite3.Cursor", "mysql.connector.cursor.MySQLCursor",
            "psycopg2.extensions.cursor", "pymysql.cursors.Cursor"]
    patterns = ["*Cursor"]
    match_subclasses = True


@python_rule(
    id="PYTHON-FLASK-SEC-003",
    name="Flask SQL Injection via Tainted String",
    severity="CRITICAL",
    category="flask",
    cwe="CWE-89",
    tags="python,flask,sql-injection,database,OWASP-A03,CWE-89",
    message="User input flows to SQL execution without parameterization. Use parameterized queries.",
    owasp="A03:2021",
)
def detect_flask_sql_injection():
    """Detects Flask request data flowing to SQL execution."""
    return flows(
        from_sources=[
            FlaskRequest.method("get", "args", "form", "values",
                                "get_json", "cookies", "headers"),
        ],
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
