from codepathfinder.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets


@python_rule(
    id="PYTHON-DJANGO-SEC-080",
    name="Django Empty Password in set_password()",
    severity="HIGH",
    category="django",
    cwe="CWE-521",
    tags="python,django,password,empty,OWASP-A07,CWE-521",
    message="Empty password set via set_password(). Use None instead of empty string.",
    owasp="A07:2021",
)
def detect_django_empty_password():
    """Audit: detects set_password() calls that may use empty strings."""
    return calls("*.set_password")
