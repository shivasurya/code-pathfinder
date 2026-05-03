from codepathfinder.python_decorators import python_rule
from codepathfinder import calls, QueryType

class CSVModule(QueryType):
    fqns = ["csv"]


@python_rule(
    id="PYTHON-LANG-SEC-094",
    name="csv.writer Without defusedcsv",
    severity="LOW",
    category="lang",
    cwe="CWE-1236",
    tags="python,csv,csv-injection,defusedcsv,CWE-1236",
    message="csv.writer() detected. Consider defusedcsv to prevent formula injection.",
    owasp="A03:2021",
)
def detect_csv_writer():
    """Detects csv.writer usage."""
    return CSVModule.method("writer", "DictWriter")
