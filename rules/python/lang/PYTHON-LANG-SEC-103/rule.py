from rules.python_decorators import python_rule
from codepathfinder import calls, QueryType

class ReModule(QueryType):
    fqns = ["re"]


@python_rule(
    id="PYTHON-LANG-SEC-103",
    name="Regex DoS Risk",
    severity="LOW",
    category="lang",
    cwe="CWE-1333",
    tags="python,regex,redos,denial-of-service,CWE-1333",
    message="re.compile/match/search detected. Audit regex patterns for catastrophic backtracking.",
    owasp="A06:2021",
)
def detect_regex_dos():
    """Detects re.compile/match/search calls — audit for regex DoS."""
    return ReModule.method("compile", "match", "search", "findall")
