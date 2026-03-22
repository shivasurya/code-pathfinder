from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets


@python_rule(
    id="PYTHON-LANG-SEC-084",
    name="Formatted SQL Query",
    severity="HIGH",
    category="lang",
    cwe="CWE-89",
    tags="python,sql-injection,formatted-query,CWE-89",
    message="SQL query built with string formatting detected. Use parameterized queries instead.",
    owasp="A03:2021",
)
def detect_formatted_sql():
    """Detects cursor.execute() calls (audit for string formatting in SQL)."""
    return calls("cursor.execute", "cursor.executemany",
                 "connection.execute", "conn.execute")
