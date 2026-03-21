from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class PickleModule(QueryType):
    fqns = ["pickle", "_pickle"]

class YamlModule(QueryType):
    fqns = ["yaml"]

class MarshalModule(QueryType):
    fqns = ["marshal"]


@python_rule(
    id="PYTHON-FLASK-SEC-013",
    name="Flask Insecure Deserialization",
    severity="CRITICAL",
    category="flask",
    cwe="CWE-502",
    tags="python,flask,deserialization,pickle,yaml,rce,owasp-a08,cwe-502",
    message="User input flows to unsafe deserialization (pickle/yaml). Use json.loads() or yaml.safe_load().",
    owasp="A08:2021",
)
def detect_flask_insecure_deserialization():
    """Detects Flask request data flowing to pickle.loads() or yaml.load()."""
    return flows(
        from_sources=[
            calls("request.get_data"),
            calls("request.get_json"),
            calls("request.form.get"),
            calls("request.args.get"),
        ],
        to_sinks=[
            PickleModule.method("loads", "load").tracks(0),
            YamlModule.method("load", "unsafe_load").tracks(0),
            MarshalModule.method("loads", "load").tracks(0),
            calls("jsonpickle.decode"),
            calls("shelve.open"),
        ],
        sanitized_by=[
            calls("json.loads"),
            calls("yaml.safe_load"),
            calls("hmac.compare_digest"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
