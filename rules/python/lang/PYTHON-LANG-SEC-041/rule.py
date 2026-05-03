from codepathfinder.python_decorators import python_rule
from codepathfinder import calls, QueryType

class YamlModule(QueryType):
    fqns = ["yaml"]


@python_rule(
    id="PYTHON-LANG-SEC-041",
    name="PyYAML Unsafe Load",
    severity="HIGH",
    category="lang",
    cwe="CWE-502",
    tags="python,yaml,deserialization,rce,OWASP-A08,CWE-502",
    message="yaml.load() or yaml.unsafe_load() detected. Use yaml.safe_load() instead.",
    owasp="A08:2021",
)
def detect_yaml_load():
    """Detects yaml.load() and yaml.unsafe_load() calls."""
    return YamlModule.method("load", "unsafe_load")
