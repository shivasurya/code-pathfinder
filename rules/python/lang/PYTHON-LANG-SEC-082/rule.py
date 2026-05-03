from codepathfinder.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class AiopgCursor(QueryType):
    fqns = ["aiopg.Cursor", "aiopg.cursor.Cursor"]
    patterns = ["*Cursor"]
    match_subclasses = True


@python_rule(
    id="PYTHON-LANG-SEC-082",
    name="aiopg SQL Injection",
    severity="CRITICAL",
    category="lang",
    cwe="CWE-89",
    tags="python,aiopg,sql-injection,database,CWE-89",
    message="String formatting in aiopg query. Use parameterized queries.",
    owasp="A03:2021",
)
def detect_aiopg_sqli():
    """Detects potential SQL injection in aiopg cursor.execute()."""
    return AiopgCursor.method("execute", "executemany")
