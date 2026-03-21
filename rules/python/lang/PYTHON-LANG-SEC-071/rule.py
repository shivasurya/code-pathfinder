from rules.python_decorators import python_rule
from codepathfinder import calls, QueryType

class ParamikoModule(QueryType):
    fqns = ["paramiko"]


@python_rule(
    id="PYTHON-LANG-SEC-071",
    name="Paramiko Implicit Trust Host Key",
    severity="HIGH",
    category="lang",
    cwe="CWE-322",
    tags="python,paramiko,ssh,host-key,mitm,cwe-322",
    message="AutoAddPolicy/WarningPolicy trusts unknown host keys. Use RejectPolicy or verify keys.",
    owasp="A02:2021",
)
def detect_paramiko_trust():
    """Detects paramiko AutoAddPolicy and WarningPolicy usage."""
    return ParamikoModule.method("AutoAddPolicy", "WarningPolicy", "set_missing_host_key_policy")
