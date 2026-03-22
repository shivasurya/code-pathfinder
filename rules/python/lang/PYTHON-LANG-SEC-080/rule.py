from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class Psycopg2Cursor(QueryType):
    fqns = ["psycopg2.extensions.cursor", "psycopg2.extras.RealDictCursor"]
    patterns = ["*cursor*"]
    match_subclasses = True


@python_rule(
    id="PYTHON-LANG-SEC-080",
    name="psycopg2 SQL Injection",
    severity="CRITICAL",
    category="lang",
    cwe="CWE-89",
    tags="python,psycopg2,sql-injection,database,OWASP-A03,CWE-89",
    message="String formatting in psycopg2 query. Use parameterized queries: cursor.execute(sql, params).",
    owasp="A03:2021",
)
def detect_psycopg2_sqli():
    """Detects potential SQL injection in psycopg2 cursor.execute()."""
    return Psycopg2Cursor.method("execute", "executemany")
