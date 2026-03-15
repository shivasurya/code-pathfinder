"""
PYTHON-FLASK-SEC-001: Flask Command Injection via os.system
PYTHON-FLASK-SEC-002: Flask Command Injection via subprocess

Security Impact: CRITICAL
CWE: CWE-78 (Improper Neutralization of Special Elements used in an OS Command)
OWASP: A03:2021 - Injection

DESCRIPTION:
These rules detect OS command injection vulnerabilities in Flask applications where
user-controlled input from HTTP request parameters flows into system command execution
functions. Two distinct attack surfaces are covered:

- **SEC-001**: Detects flows to os.system(), os.popen(), and related os module functions
  that execute shell commands as a single string, making them inherently vulnerable to
  shell metacharacter injection.

- **SEC-002**: Detects flows to subprocess module functions (subprocess.call, subprocess.run,
  subprocess.Popen, etc.) where user input may be passed as part of a command string,
  especially when shell=True is used.

Command injection allows attackers to execute arbitrary operating system commands on the
server hosting the Flask application, typically leading to full system compromise.

SECURITY IMPLICATIONS:

**1. Remote Code Execution**:
Attackers inject shell metacharacters (;, |, &&, ``, $()) to chain arbitrary commands
onto the intended operation. For example, injecting `; cat /etc/shadow` after a filename
parameter.

**2. System Compromise**:
With command execution, attackers can install backdoors, create user accounts, modify
system configurations, pivot to internal networks, and establish persistent access.

**3. Data Exfiltration**:
Attackers can use commands like curl, wget, or netcat to exfiltrate sensitive files,
environment variables containing secrets, and database credentials.

**4. Lateral Movement**:
Compromised servers can be used as pivot points to attack other systems on the internal
network, access cloud metadata endpoints, or interact with adjacent services.

VULNERABLE EXAMPLE:
```python
# --- file: app.py ---
from flask import Flask, request
from utils import run_diagnostic

app = Flask(__name__)


@app.route('/diag')
def diagnostics():
    host = request.args.get('host')
    output = run_diagnostic(host)
    return output

# --- file: utils.py ---
import os
import subprocess


def run_diagnostic(target):
    cmd = "ping -c 3 " + target
    os.system(cmd)
    result = subprocess.check_output(cmd, shell=True)
    return result.decode()
```

SECURE EXAMPLE:
```python
from flask import Flask, request
import subprocess
import shlex

app = Flask(__name__)

@app.route('/ping')
def ping_host():
    host = request.args.get('host')
    # SAFE: Use subprocess with list arguments (no shell interpretation)
    result = subprocess.run(['ping', '-c', '3', host], capture_output=True, text=True)
    return result.stdout

@app.route('/lookup')
def dns_lookup():
    domain = request.args.get('domain')
    # SAFE: shlex.quote() escapes shell metacharacters
    safe_domain = shlex.quote(domain)
    result = subprocess.check_output(['nslookup', safe_domain])
    return result
```

DETECTION AND PREVENTION:

**Pre-deployment checks**:
```bash
pathfinder scan --project . --ruleset cpf/python/PYTHON-FLASK-SEC-001,PYTHON-FLASK-SEC-002
```

**Code Review Checklist**:
- [ ] No os.system() or os.popen() with user input
- [ ] subprocess calls use list arguments, not shell strings
- [ ] shell=True is never used with user-controlled data
- [ ] shlex.quote() applied to any user input passed to shell commands
- [ ] Input validation with allowlists for expected command parameters

COMPLIANCE:
- CWE-78: Improper Neutralization of Special Elements used in an OS Command
- OWASP Top 10 A03:2021 - Injection
- SANS Top 25 (CWE-78 ranked #5)
- PCI DSS Requirement 6.5.1: Injection Flaws

REFERENCES:
- CWE-78: https://cwe.mitre.org/data/definitions/78.html
- OWASP Command Injection: https://owasp.org/www-community/attacks/Command_Injection
- Python subprocess documentation: https://docs.python.org/3/library/subprocess.html
- Python shlex.quote: https://docs.python.org/3/library/shlex.html#shlex.quote

DETECTION SCOPE:
These rules perform inter-procedural taint analysis tracking data from Flask request sources
to os and subprocess sinks. Recognized sanitizers include shlex.quote() and shlex.split().
"""

from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets


class OSModule(QueryType):
    fqns = ["os"]


class SubprocessModule(QueryType):
    fqns = ["subprocess"]


@python_rule(
    id="PYTHON-FLASK-SEC-001",
    name="Flask Command Injection via os.system",
    severity="CRITICAL",
    category="flask",
    cwe="CWE-78",
    tags="python,flask,command-injection,os-system,owasp-a03,cwe-78",
    message="User input flows to os.system(). Use subprocess with list args and shlex.quote() instead.",
    owasp="A03:2021",
)
def detect_flask_os_system_injection():
    """Detects Flask request data flowing to os.system() or os.popen()."""
    return flows(
        from_sources=[
            calls("request.args.get"),
            calls("request.form.get"),
            calls("request.values.get"),
            calls("request.get_json"),
            calls("request.cookies.get"),
            calls("request.headers.get"),
        ],
        to_sinks=[
            OSModule.method("system", "popen", "popen2", "popen3", "popen4").tracks(0),
        ],
        sanitized_by=[
            calls("shlex.quote"),
            calls("shlex.split"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )


@python_rule(
    id="PYTHON-FLASK-SEC-002",
    name="Flask Command Injection via subprocess",
    severity="CRITICAL",
    category="flask",
    cwe="CWE-78",
    tags="python,flask,command-injection,subprocess,owasp-a03,cwe-78",
    message="User input flows to subprocess call. Use shlex.quote() or avoid shell=True.",
    owasp="A03:2021",
)
def detect_flask_subprocess_injection():
    """Detects Flask request data flowing to subprocess functions."""
    return flows(
        from_sources=[
            calls("request.args.get"),
            calls("request.form.get"),
            calls("request.values.get"),
            calls("request.get_json"),
            calls("request.cookies.get"),
            calls("request.headers.get"),
        ],
        to_sinks=[
            SubprocessModule.method("call", "check_call", "check_output",
                                    "run", "Popen", "getoutput", "getstatusoutput").tracks(0),
        ],
        sanitized_by=[
            calls("shlex.quote"),
            calls("shlex.split"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
