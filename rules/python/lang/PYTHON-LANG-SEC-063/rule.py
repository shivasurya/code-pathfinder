from rules.python_decorators import python_rule
from codepathfinder import calls, QueryType

class FtplibModule(QueryType):
    fqns = ["ftplib"]


@python_rule(
    id="PYTHON-LANG-SEC-063",
    name="FTP Without TLS",
    severity="MEDIUM",
    category="lang",
    cwe="CWE-319",
    tags="python,ftp,insecure-transport,CWE-319",
    message="ftplib.FTP() without TLS. Use ftplib.FTP_TLS() instead.",
    owasp="A02:2021",
)
def detect_ftp_no_tls():
    """Detects ftplib.FTP usage without TLS."""
    return FtplibModule.method("FTP")
