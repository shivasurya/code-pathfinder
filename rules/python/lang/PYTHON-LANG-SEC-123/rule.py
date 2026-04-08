from rules.python_decorators import python_rule
from codepathfinder import calls, QueryType


@python_rule(
    id="PYTHON-LANG-SEC-123",
    name="Jinja2 Server-Side Template Injection",
    severity="HIGH",
    category="lang",
    cwe="CWE-1336",
    tags="python,jinja2,ssti,template-injection,rce,CWE-1336,OWASP-A03",
    message="Jinja2 template constructed from dynamic input detected. from_string(), Template(), and render_template_string() can lead to server-side template injection (SSTI).",
    owasp="A03:2021",
)
def detect_jinja2_ssti():
    """Detects Jinja2 SSTI-prone template construction from dynamic input."""
    return calls("*.from_string", "jinja2.Template", "Template", "render_template_string")
