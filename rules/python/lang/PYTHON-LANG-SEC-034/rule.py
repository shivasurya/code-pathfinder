from rules.python_decorators import python_rule
from codepathfinder import calls, QueryType

class HashlibModule(QueryType):
    fqns = ["hashlib"]


@python_rule(
    id="PYTHON-LANG-SEC-034",
    name="MD5 Used for Password Hashing",
    severity="HIGH",
    category="lang",
    cwe="CWE-327",
    tags="python,md5,password,weak-hash,cwe-327",
    message="MD5 used for password hashing. Use bcrypt, scrypt, or argon2 instead.",
    owasp="A02:2021",
)
def detect_md5_password():
    """Detects MD5 used in password context -- audit-level detection."""
    return HashlibModule.method("md5")
