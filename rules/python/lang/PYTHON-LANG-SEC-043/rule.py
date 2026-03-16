from rules.python_decorators import python_rule
from codepathfinder import calls, QueryType

class YamlModule(QueryType):
    fqns = ["yaml"]

class RuamelYamlModule(QueryType):
    fqns = ["ruamel.yaml"]


@python_rule(
    id="PYTHON-LANG-SEC-043",
    name="ruamel.yaml Unsafe Usage",
    severity="HIGH",
    category="lang",
    cwe="CWE-502",
    tags="python,ruamel,yaml,deserialization,rce,cwe-502",
    message="ruamel.yaml with unsafe typ detected. Use typ='safe' instead.",
    owasp="A08:2021",
)
def detect_ruamel_unsafe():
    """Detects ruamel.yaml YAML() with unsafe typ."""
    return RuamelYamlModule.method("YAML").where("typ", "unsafe")
