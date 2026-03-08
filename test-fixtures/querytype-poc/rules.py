"""QueryType PoC rules — SQL injection and weak hash detection."""

from codepathfinder import rule, QueryType, Or, lt, gt, lte, gte, regex, missing
from codepathfinder.dataflow import flows
from codepathfinder.matchers import calls


# --- QueryType definitions ---

class DBCursor(QueryType):
    fqns = ["sqlite3.Cursor", "mysql.connector.cursor.MySQLCursor"]
    patterns = ["*Cursor"]
    match_subclasses = True


class WebRequest(QueryType):
    fqns = ["flask.Request", "django.http.HttpRequest"]
    match_subclasses = True


class Hashlib(QueryType):
    fqns = ["hashlib"]


class OSModule(QueryType):
    fqns = ["os"]


# --- Rules ---

@rule(id="SQL-INJECTION-POC", severity="critical", cwe="CWE-89")
def sql_injection_poc():
    return flows(
        from_sources=WebRequest.method("get", "args"),
        to_sinks=DBCursor.method("execute"),
    )


@rule(id="SQL-INJECTION-FALLBACK", severity="critical", cwe="CWE-89")
def sql_injection_fallback():
    """Same rule but with fallbackMode=name to catch without type inference."""
    return flows(
        from_sources=calls("request.args.get"),
        to_sinks=calls("cursor.execute"),
    )


@rule(id="WEAK-HASH-POC", severity="medium", cwe="CWE-327")
def weak_hash():
    return Or(Hashlib.method("md5"), Hashlib.method("sha1"))


@rule(id="OVERLY-PERMISSIVE-FILE", severity="high", cwe="CWE-732")
def overly_permissive():
    return OSModule.method("chmod").arg(1, "0o7*")
