from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class AsyncpgConnection(QueryType):
    fqns = ["asyncpg.Connection", "asyncpg.connection.Connection"]
    patterns = ["*Connection"]
    match_subclasses = True


@python_rule(
    id="PYTHON-LANG-SEC-081",
    name="asyncpg SQL Injection",
    severity="CRITICAL",
    category="lang",
    cwe="CWE-89",
    tags="python,asyncpg,sql-injection,database,CWE-89",
    message="String formatting in asyncpg query. Use parameterized queries with $1 placeholders.",
    owasp="A03:2021",
)
def detect_asyncpg_sqli():
    """Detects potential SQL injection in asyncpg connection methods."""
    return AsyncpgConnection.method("execute", "executemany", "fetch", "fetchrow", "fetchval")
