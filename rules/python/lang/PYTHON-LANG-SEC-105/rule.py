from rules.python_decorators import python_rule
from codepathfinder import calls, QueryType

class LoggingModule(QueryType):
    fqns = ["logging"]


@python_rule(
    id="PYTHON-LANG-SEC-105",
    name="Logger Credential Leak Risk",
    severity="MEDIUM",
    category="lang",
    cwe="CWE-532",
    tags="python,logging,credentials,information-disclosure,cwe-532",
    message="Logging call detected. Audit log statements for credential/secret leakage.",
    owasp="A09:2021",
)
def detect_logger_cred_leak():
    """Detects logging calls — audit for credential leakage."""
    return LoggingModule.method("info", "debug", "warning", "error", "critical", "exception", "log")
