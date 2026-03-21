from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class Builtins(QueryType):
    fqns = ["builtins"]

class IOModule(QueryType):
    fqns = ["io"]


@python_rule(
    id="PYTHON-FLASK-SEC-007",
    name="Flask Path Traversal via open()",
    severity="HIGH",
    category="flask",
    cwe="CWE-22",
    tags="python,flask,path-traversal,file-access,owasp-a01,cwe-22",
    message="User input flows to open(). Use os.path.basename() or werkzeug.utils.secure_filename().",
    owasp="A01:2021",
)
def detect_flask_path_traversal():
    """Detects Flask request data flowing to file open()."""
    return flows(
        from_sources=[
            calls("request.args.get"),
            calls("request.form.get"),
            calls("request.values.get"),
            calls("request.get_json"),
        ],
        to_sinks=[
            Builtins.method("open").tracks(0),
            IOModule.method("open").tracks(0),
            calls("open"),
        ],
        sanitized_by=[
            calls("os.path.basename"),
            calls("secure_filename"),
            calls("werkzeug.utils.secure_filename"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
