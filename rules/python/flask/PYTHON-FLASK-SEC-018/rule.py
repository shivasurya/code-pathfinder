from rules.python_decorators import python_rule
from codepathfinder import calls, Or, QueryType

class HashidsModule(QueryType):
    fqns = ["hashids"]


@python_rule(
    id="PYTHON-FLASK-SEC-018",
    name="Flask Hashids with Secret Key",
    severity="MEDIUM",
    category="flask",
    cwe="CWE-330",
    tags="python,flask,hashids,secret-key,cwe-330",
    message="Flask SECRET_KEY used as Hashids salt. Use a separate salt value.",
    owasp="A02:2021",
)
def detect_flask_hashids_secret():
    """Detects Hashids(salt=app.secret_key)."""
    return HashidsModule.method("Hashids").where("salt", "app.secret_key")
