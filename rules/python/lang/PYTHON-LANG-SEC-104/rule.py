from codepathfinder.python_decorators import python_rule
from codepathfinder import calls, QueryType

class LoggingConfig(QueryType):
    fqns = ["logging.config"]


@python_rule(
    id="PYTHON-LANG-SEC-104",
    name="logging.config.listen() Eval Risk",
    severity="HIGH",
    category="lang",
    cwe="CWE-95",
    tags="python,logging,listen,eval,code-execution,CWE-95",
    message="logging.config.listen() can execute arbitrary code via configuration. Restrict access.",
    owasp="A03:2021",
)
def detect_logging_listen():
    """Detects logging.config.listen() which evaluates received config."""
    return LoggingConfig.method("listen")
