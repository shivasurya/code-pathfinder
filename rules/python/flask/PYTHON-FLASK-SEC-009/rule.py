from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class CSVWriter(QueryType):
    fqns = ["csv.writer", "csv.DictWriter"]
    patterns = ["*Writer"]
    match_subclasses = True


@python_rule(
    id="PYTHON-FLASK-SEC-009",
    name="Flask CSV Injection",
    severity="MEDIUM",
    category="flask",
    cwe="CWE-1236",
    tags="python,flask,csv-injection,CWE-1236",
    message="User input flows to CSV writer. Sanitize by removing leading =, +, -, @ characters.",
    owasp="A03:2021",
)
def detect_flask_csv_injection():
    """Detects Flask request data flowing to csv.writer.writerow()."""
    return flows(
        from_sources=[
            calls("request.args.get"),
            calls("request.form.get"),
            calls("request.values.get"),
            calls("request.get_json"),
        ],
        to_sinks=[
            CSVWriter.method("writerow", "writerows").tracks(0),
            calls("writer.writerow"),
            calls("writer.writerows"),
            calls("csv.writer"),
        ],
        sanitized_by=[],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
