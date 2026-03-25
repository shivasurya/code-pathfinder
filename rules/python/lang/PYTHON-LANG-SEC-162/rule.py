from rules.python_decorators import python_rule
from codepathfinder import calls, QueryType


@python_rule(
    id="PYTHON-LANG-SEC-162",
    name="Symlink Following Arbitrary File Access",
    severity="HIGH",
    category="lang",
    cwe="CWE-59",
    tags="python,symlink,path-traversal,file-access,OWASP-A01,CWE-59",
    message="Symlink operation detected. Symlink following on user-controlled paths can lead to arbitrary file read/write.",
    owasp="A01:2021",
)
def detect_symlink_following():
    """Detects os.readlink(), os.symlink(), is_symlink(), and os.path.islink() calls."""
    return calls("os.readlink", "os.symlink", "*.is_symlink", "os.path.islink")
