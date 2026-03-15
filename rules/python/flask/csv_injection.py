"""
PYTHON-FLASK-SEC-009: Flask CSV Injection
PYTHON-FLASK-SEC-010: Flask NaN Injection
PYTHON-FLASK-SEC-011: Flask Tainted URL Host

Security Impact: MEDIUM (SEC-009, SEC-011), LOW (SEC-010)
CWE: CWE-1236 (CSV Injection), CWE-704 (Incorrect Type Conversion or Cast), CWE-918 (SSRF)
OWASP: A03:2021 - Injection, A10:2021 - SSRF

DESCRIPTION:
This module contains three distinct injection detection rules for Flask applications targeting
different attack surfaces:

- **SEC-009 (CSV Injection / CWE-1236)**: Detects user input flowing into CSV writer
  functions (csv.writer.writerow, csv.writer.writerows). When user-controlled data is
  written to CSV files without sanitization, attackers can inject spreadsheet formula
  payloads (beginning with =, +, -, or @) that execute when the CSV is opened in
  spreadsheet applications like Excel or Google Sheets. This can lead to data exfiltration
  via formula-based HTTP requests or arbitrary command execution through DDE (Dynamic Data
  Exchange) on Windows systems.

- **SEC-010 (NaN Injection / CWE-704)**: Detects user input flowing to float() type
  conversion. Python's float() accepts special string values "nan", "inf", and "-inf"
  which produce IEEE 754 special values. These values have unexpected comparison behavior
  (NaN != NaN evaluates to True, NaN comparisons always return False) which can break
  business logic, bypass validation checks, cause division-by-zero with infinity, and
  corrupt numerical data processing pipelines.

- **SEC-011 (Tainted URL Host / CWE-918)**: Detects user input used in constructing URL
  host portions that flow to HTTP request libraries. This is a variant of SSRF where the
  attacker specifically controls the hostname component of a URL, allowing them to redirect
  server-side requests to internal services, cloud metadata endpoints, or attacker-controlled
  servers for credential theft.

SECURITY IMPLICATIONS:

**CSV Injection (SEC-009)**:
- Spreadsheet formula execution in victim's application
- Data exfiltration via HYPERLINK() or WEBSERVICE() formulas
- DDE-based command execution on Windows systems
- Social engineering via modified spreadsheet content

**NaN Injection (SEC-010)**:
- Bypass of numeric range validation (NaN fails all comparisons)
- Corruption of aggregation functions (sum, average) with NaN propagation
- Infinite loop conditions with infinity values
- Financial calculation errors leading to incorrect transactions

**Tainted URL Host (SEC-011)**:
- Access to internal services and cloud metadata endpoints
- Credential theft from redirected authentication flows
- Port scanning of internal networks
- Data exfiltration through DNS or HTTP to attacker hosts

VULNERABLE EXAMPLE:
```python
from flask import Flask, request
import csv
import io

app = Flask(__name__)

@app.route('/export')
def export_csv():
    name = request.args.get('name')
    output = io.StringIO()
    writer = csv.writer(output)
    writer.writerow([name, "data"])
    return output.getvalue()
```

SECURE EXAMPLE:
```python
from flask import Flask, request
import csv, io, requests, math

app = Flask(__name__)

def sanitize_csv_value(value):
    \"\"\"Strip leading formula characters from CSV values.\"\"\"
    if isinstance(value, str) and value and value[0] in ('=', '+', '-', '@'):
        return "'" + value  # Prefix with single quote to prevent formula execution
    return value

@app.route('/export')
def export_csv():
    name = request.args.get('name')
    writer = csv.writer(io.StringIO())
    # SAFE: Sanitize before writing to CSV
    writer.writerow([sanitize_csv_value(name), 'data'])

@app.route('/convert')
def convert():
    value = request.args.get('value')
    num = float(value)
    # SAFE: Reject NaN and Inf values
    if math.isnan(num) or math.isinf(num):
        return {'error': 'Invalid number'}, 400
    return {'result': num}

ALLOWED_HOSTS = {'api.example.com', 'cdn.example.com'}

@app.route('/proxy')
def proxy():
    host = request.args.get('host')
    # SAFE: Validate host against allowlist
    if host not in ALLOWED_HOSTS:
        return {'error': 'Host not allowed'}, 403
    response = requests.get(f'https://{host}/api/data')
    return response.text
```

DETECTION AND PREVENTION:

**Pre-deployment checks**:
```bash
pathfinder scan --project . --ruleset cpf/python/PYTHON-FLASK-SEC-009,PYTHON-FLASK-SEC-010,PYTHON-FLASK-SEC-011
```

**Code Review Checklist**:
- [ ] CSV output sanitizes leading formula characters (=, +, -, @)
- [ ] Consider using defusedcsv as a drop-in replacement for csv module
- [ ] float() results checked with math.isnan() and math.isinf()
- [ ] URL host components validated against allowlists
- [ ] Private/reserved IP ranges blocked in URL construction

COMPLIANCE:
- CWE-1236: Improper Neutralization of Formula Elements in a CSV File
- CWE-704: Incorrect Type Conversion or Cast
- CWE-918: Server-Side Request Forgery (SSRF)
- OWASP Top 10 A03:2021 - Injection
- OWASP Top 10 A10:2021 - Server-Side Request Forgery

REFERENCES:
- CWE-1236: https://cwe.mitre.org/data/definitions/1236.html
- CWE-704: https://cwe.mitre.org/data/definitions/704.html
- CWE-918: https://cwe.mitre.org/data/definitions/918.html
- OWASP CSV Injection: https://owasp.org/www-community/attacks/CSV_Injection
- OWASP SSRF Prevention: https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html
- defusedcsv library: https://github.com/raphaelm/defusedcsv

DETECTION SCOPE:
These rules perform inter-procedural taint analysis tracking data from Flask request sources
to their respective sinks. SEC-009 tracks to csv.writer methods with no recognized sanitizers.
SEC-010 tracks to float() with int() as a sanitizer. SEC-011 tracks to HTTP request libraries
with validate_host() and is_safe_url() as sanitizers.
"""

from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets


class CSVWriter(QueryType):
    fqns = ["csv.writer", "csv.DictWriter"]
    patterns = ["*Writer"]
    match_subclasses = True


class RequestsLib(QueryType):
    fqns = ["requests"]


class Builtins(QueryType):
    fqns = ["builtins"]


@python_rule(
    id="PYTHON-FLASK-SEC-009",
    name="Flask CSV Injection",
    severity="MEDIUM",
    category="flask",
    cwe="CWE-1236",
    tags="python,flask,csv-injection,cwe-1236",
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


@python_rule(
    id="PYTHON-FLASK-SEC-010",
    name="Flask NaN Injection",
    severity="LOW",
    category="flask",
    cwe="CWE-704",
    tags="python,flask,nan-injection,type-confusion,cwe-704",
    message="User input flows to float() which may produce NaN/Inf. Validate numeric input.",
    owasp="A03:2021",
)
def detect_flask_nan_injection():
    """Detects Flask request data flowing to float() conversion."""
    return flows(
        from_sources=[
            calls("request.args.get"),
            calls("request.form.get"),
            calls("request.values.get"),
        ],
        to_sinks=[
            Builtins.method("float").tracks(0),
            calls("float"),
        ],
        sanitized_by=[
            calls("int"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )


@python_rule(
    id="PYTHON-FLASK-SEC-011",
    name="Flask Tainted URL Host",
    severity="HIGH",
    category="flask",
    cwe="CWE-918",
    tags="python,flask,ssrf,url-host,owasp-a10,cwe-918",
    message="User input used in URL host construction. Validate against an allowlist of hosts.",
    owasp="A10:2021",
)
def detect_flask_tainted_url_host():
    """Detects Flask request data used in URL host construction flowing to HTTP requests."""
    return flows(
        from_sources=[
            calls("request.args.get"),
            calls("request.form.get"),
            calls("request.values.get"),
            calls("request.get_json"),
        ],
        to_sinks=[
            RequestsLib.method("get", "post", "put", "delete").tracks(0),
            calls("http_requests.get"),
            calls("http_requests.post"),
            calls("urllib.request.urlopen"),
        ],
        sanitized_by=[
            calls("*.validate_host"),
            calls("*.is_safe_url"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
