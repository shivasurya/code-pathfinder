from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class Pg8000Cursor(QueryType):
    fqns = ["pg8000.Cursor", "pg8000.core.Cursor"]
    patterns = ["*Cursor"]
    match_subclasses = True


@python_rule(
    id="PYTHON-LANG-SEC-083",
    name="pg8000 SQL Injection",
    severity="CRITICAL",
    category="lang",
    cwe="CWE-89",
    tags="python,pg8000,sql-injection,database,CWE-89",
    message="String formatting in pg8000 query. Use parameterized queries.",
    owasp="A03:2021",
)
def detect_pg8000_sqli():
    """Detects potential SQL injection in pg8000 cursor.execute()."""
    return Pg8000Cursor.method("execute", "executemany")
