from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class SQLAlchemySession(QueryType):
    fqns = ["sqlalchemy.orm.Session", "sqlalchemy.orm.session.Session"]
    patterns = ["*Session"]
    match_subclasses = True

_LAMBDA_SOURCES = [
    calls("event.get"),
    calls("event.items"),
    calls("event.values"),
    calls("*.get"),
]


@python_rule(
    id="PYTHON-LAMBDA-SEC-014",
    name="Lambda SQL Injection via SQLAlchemy",
    severity="CRITICAL",
    category="aws_lambda",
    cwe="CWE-89",
    tags="python,aws,lambda,sql-injection,sqlalchemy,OWASP-A03,CWE-89",
    message="Lambda event data flows to SQLAlchemy session.execute(). Use text() with params.",
    owasp="A03:2021",
)
def detect_lambda_sqlalchemy_sqli():
    """Detects Lambda event data flowing to SQLAlchemy session.execute()."""
    return flows(
        from_sources=_LAMBDA_SOURCES,
        to_sinks=[
            SQLAlchemySession.method("execute").tracks(0),
            calls("session.execute"),
            calls("*.execute"),
        ],
        sanitized_by=[
            calls("text"),
            calls("sqlalchemy.text"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
