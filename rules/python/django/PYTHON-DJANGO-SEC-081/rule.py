from codepathfinder.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets


@python_rule(
    id="PYTHON-DJANGO-SEC-081",
    name="Django Default Empty Password Value",
    severity="HIGH",
    category="django",
    cwe="CWE-521",
    tags="python,django,password,default,OWASP-A07,CWE-521",
    message="Password default value may be empty string. Use None as default.",
    owasp="A07:2021",
)
def detect_django_default_empty_password():
    """Audit: detects request.POST.get with potential empty password default flowing to set_password."""
    return flows(
        from_sources=[
            calls("request.POST.get"),
            calls("*.POST.get"),
        ],
        to_sinks=[
            calls("*.set_password"),
        ],
        sanitized_by=[],
        propagates_through=PropagationPresets.standard(),
        scope="local",
    )
