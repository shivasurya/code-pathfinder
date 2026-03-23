from rules.python_decorators import python_rule
from codepathfinder import calls, QueryType


@python_rule(
    id="PYTHON-LANG-SEC-102",
    name="Hardcoded Password in Default Argument",
    severity="HIGH",
    category="lang",
    cwe="CWE-259",
    tags="python,hardcoded-password,credentials,CWE-259",
    message="Hardcoded password detected in function default argument. Use environment variables or secrets manager.",
    owasp="A07:2021",
)
def detect_hardcoded_password():
    """Detects functions with password-like default arguments — audit level."""
    return calls("*.connect", "*.login", "*.authenticate",
                 match_name={"password": "*"})
