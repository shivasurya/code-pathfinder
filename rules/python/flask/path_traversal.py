"""
PYTHON-FLASK-SEC-007: Flask Path Traversal via open()

Security Impact: HIGH
CWE: CWE-22 (Improper Limitation of a Pathname to a Restricted Directory)
OWASP: A01:2021 - Broken Access Control

DESCRIPTION:
This rule detects path traversal vulnerabilities in Flask applications where user-controlled
input flows into file system open() calls without proper sanitization. When an attacker
supplies directory traversal sequences such as `../` in a filename parameter, they can
escape the intended directory and access arbitrary files on the server's file system.

Path traversal is a fundamental file system access control vulnerability. In Flask web
applications, it commonly occurs when route handlers accept user-supplied filenames or
paths from request parameters and pass them directly to Python's built-in open() function
or io.open() without stripping or validating the path components.

SECURITY IMPLICATIONS:

**1. Sensitive File Disclosure**:
Attackers can read configuration files (/etc/passwd, /etc/shadow), application source
code, database files, environment files (.env), and private keys by traversing out of
the web root directory.

**2. Application Secret Exposure**:
Flask's secret key, database connection strings, API keys, and other credentials stored
in configuration files or environment files can be directly read.

**3. Source Code Theft**:
Complete application source code can be downloaded by traversing to the application
directory, revealing business logic, additional vulnerabilities, and hardcoded secrets.

**4. Arbitrary File Write (if open() used for writing)**:
When the vulnerable open() call uses write mode, attackers can overwrite configuration
files, inject malicious code into Python files, or write web shells to accessible
directories.

VULNERABLE EXAMPLE:
```python
from flask import Flask, request

app = Flask(__name__)

@app.route('/download')
def download_file():
    filename = request.args.get('file')
    # VULNERABLE: User controls the file path
    with open('/uploads/' + filename, 'r') as f:
        return f.read()

# Attack: GET /download?file=../../../etc/passwd
# Reads /uploads/../../../etc/passwd = /etc/passwd
```

SECURE EXAMPLE:
```python
from flask import Flask, request, abort
from werkzeug.utils import secure_filename
import os

app = Flask(__name__)
UPLOAD_DIR = '/uploads'

@app.route('/download')
def download_file():
    filename = request.args.get('file')
    # SAFE: secure_filename strips directory separators and traversal sequences
    safe_name = secure_filename(filename)
    filepath = os.path.join(UPLOAD_DIR, safe_name)

    # Additional check: ensure resolved path is within allowed directory
    if not os.path.realpath(filepath).startswith(os.path.realpath(UPLOAD_DIR)):
        abort(403)

    with open(filepath, 'r') as f:
        return f.read()
```

DETECTION AND PREVENTION:

**Pre-deployment checks**:
```bash
pathfinder scan --project . --ruleset cpf/python/PYTHON-FLASK-SEC-007
```

**Code Review Checklist**:
- [ ] No user input flows directly to open() or io.open()
- [ ] werkzeug.utils.secure_filename() applied to user-supplied filenames
- [ ] os.path.realpath() used to resolve symlinks before path validation
- [ ] Resolved path verified to stay within the intended base directory
- [ ] File access permissions follow principle of least privilege

COMPLIANCE:
- CWE-22: Improper Limitation of a Pathname to a Restricted Directory
- OWASP Top 10 A01:2021 - Broken Access Control
- SANS Top 25 (CWE-22 ranked #8)
- PCI DSS Requirement 6.5.8: Improper Access Control

REFERENCES:
- CWE-22: https://cwe.mitre.org/data/definitions/22.html
- OWASP Path Traversal: https://owasp.org/www-community/attacks/Path_Traversal
- Werkzeug secure_filename: https://werkzeug.palletsprojects.com/en/latest/utils/#werkzeug.utils.secure_filename
- Flask File Uploads: https://flask.palletsprojects.com/en/latest/patterns/fileuploads/

DETECTION SCOPE:
This rule performs inter-procedural taint analysis tracking data from Flask request sources
to open() and io.open() sinks. Recognized sanitizers include os.path.basename(),
secure_filename(), and werkzeug.utils.secure_filename().
"""

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
