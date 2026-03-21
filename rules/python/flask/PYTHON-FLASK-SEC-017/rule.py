from rules.python_decorators import python_rule
from codepathfinder import calls, Or, QueryType


@python_rule(
    id="PYTHON-FLASK-SEC-017",
    name="Flask Insecure Static File Serve",
    severity="MEDIUM",
    category="flask",
    cwe="CWE-22",
    tags="python,flask,path-traversal,static-files,cwe-22",
    message="send_from_directory() with user input. Use werkzeug.utils.secure_filename().",
    owasp="A01:2021",
)
def detect_flask_insecure_static_serve():
    """Detects send_from_directory() usage (audit for user-controlled filename)."""
    return Or(
        calls("send_from_directory"),
        calls("flask.send_from_directory"),
        calls("send_file"),
        calls("flask.send_file"),
    )
