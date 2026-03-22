from rules.python_decorators import python_rule
from codepathfinder import calls, Or


@python_rule(
    id="PYTHON-FLASK-001",
    name="Flask Debug Mode Enabled",
    severity="HIGH",
    category="flask",
    cwe="CWE-489",
    cve="CVE-2015-5306",
    tags="python,flask,debug-mode,configuration,information-disclosure,OWASP-A05,CWE-489,production,werkzeug,security,misconfiguration",
    message="Flask debug mode enabled. Never use debug=True in production. Use a production WSGI server like Gunicorn.",
    owasp="A05:2021",
)
def detect_flask_debug_mode():
    """
    Detects Flask applications with debug mode enabled.

    Matches:
    - app.run(debug=True)
    - *.run(debug=True)

    Example vulnerable code:
        app = Flask(__name__)
        app.run(debug=True)  # Detected!
    """
    from codepathfinder import QueryType

    class FlaskApp(QueryType):
        fqns = ["flask"]

    return FlaskApp.method("run").where("debug", True)
