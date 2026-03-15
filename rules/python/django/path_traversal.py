"""
Python Django Path Traversal Rules

Rules:
- PYTHON-DJANGO-SEC-040: Path Traversal via open() (CWE-22)
- PYTHON-DJANGO-SEC-041: Path Traversal via os.path.join() (CWE-22)

Security Impact: HIGH
CWE: CWE-22 (Improper Limitation of a Pathname to a Restricted Directory)
OWASP: A01:2021 - Broken Access Control

DESCRIPTION:
These rules detect path traversal vulnerabilities in Django applications where untrusted
user input from HTTP requests flows into file system operations such as open(), os.path.join(),
or other file access functions. Path traversal (also known as directory traversal) occurs when
user-controlled input containing sequences like "../" is used to construct file paths, allowing
attackers to escape the intended directory and access arbitrary files on the server's file system.

SECURITY IMPLICATIONS:

**1. Sensitive File Disclosure**:
Attackers can read sensitive server files such as /etc/passwd, /etc/shadow, application
configuration files containing database credentials, API keys in .env files, or source
code files by traversing outside the intended directory.

**2. Source Code Exposure**:
By navigating to application directories, attackers can read source code to discover
additional vulnerabilities, hardcoded secrets, internal API endpoints, and business logic
that should remain confidential.

**3. Arbitrary File Write**:
If user input reaches file write operations (open() with 'w' mode), attackers can overwrite
configuration files, inject malicious code into application files, or create web shells for
persistent access.

**4. Application Configuration Tampering**:
Attackers may overwrite Django settings files, modify URL configurations, or alter template
files to inject malicious content served to all users.

VULNERABLE EXAMPLE:
```python
import os


# SEC-040: path traversal via open
def vulnerable_open(request):
    filename = request.GET.get('file')
    with open(filename) as f:
        return f.read()


# SEC-041: path traversal via os.path.join
def vulnerable_path_join(request):
    user_path = request.GET.get('path')
    full_path = os.path.join('/uploads', user_path)
    with open(full_path) as f:
        return f.read()
```

SECURE EXAMPLE:
```python
from django.http import HttpResponse, FileResponse, Http404
import os

UPLOAD_DIR = '/var/www/uploads/'

def download_file(request):
    # SECURE: Validate resolved path stays within allowed directory
    filename = request.GET.get('file', '')
    # Use basename to strip directory traversal sequences
    safe_name = os.path.basename(filename)
    filepath = os.path.join(UPLOAD_DIR, safe_name)
    # Double-check with realpath
    real_path = os.path.realpath(filepath)
    if not real_path.startswith(os.path.realpath(UPLOAD_DIR)):
        raise Http404("File not found")
    if not os.path.isfile(real_path):
        raise Http404("File not found")
    return FileResponse(open(real_path, 'rb'))

def read_template(request):
    # SECURE: Allowlist of permitted templates
    ALLOWED_TEMPLATES = {'header.html', 'footer.html', 'sidebar.html'}
    template = request.GET.get('template', '')
    if template not in ALLOWED_TEMPLATES:
        raise Http404("Template not found")
    filepath = os.path.join('/app/templates/', template)
    with open(filepath) as f:
        return HttpResponse(f.read())
```

DETECTION AND PREVENTION:

**Key Mitigation Strategies**:
- Use os.path.basename() to strip directory components from user input
- Validate resolved paths with os.path.realpath() to ensure they stay within allowed directories
- Maintain an allowlist of permitted filenames or file patterns
- Never concatenate user input directly into file paths
- Use Django's built-in static file serving and template loading mechanisms
- Set restrictive file system permissions so the application process cannot access sensitive files
- Consider using a dedicated file storage service (S3, GCS) instead of local file paths

**Pre-deployment checks**:
```bash
pathfinder scan --project . --ruleset cpf/python/django/path-traversal
```

COMPLIANCE:
- CWE-22: Improper Limitation of a Pathname to a Restricted Directory
- OWASP A01:2021 - Broken Access Control
- SANS Top 25: CWE-22 ranked #8
- NIST SP 800-53: AC-3 (Access Enforcement)

REFERENCES:
- CWE-22: https://cwe.mitre.org/data/definitions/22.html
- OWASP Path Traversal: https://owasp.org/www-community/attacks/Path_Traversal
- Python os.path documentation: https://docs.python.org/3/library/os.path.html
- Django Security: https://docs.djangoproject.com/en/stable/topics/security/
"""

from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets


class IOModule(QueryType):
    fqns = ["io"]


_DJANGO_SOURCES = [
    calls("request.GET.get"),
    calls("request.POST.get"),
    calls("request.GET"),
    calls("request.POST"),
    calls("request.COOKIES.get"),
    calls("request.FILES.get"),
    calls("*.GET.get"),
    calls("*.POST.get"),
]


@python_rule(
    id="PYTHON-DJANGO-SEC-040",
    name="Django Path Traversal via open()",
    severity="HIGH",
    category="django",
    cwe="CWE-22",
    tags="python,django,path-traversal,open,owasp-a01,cwe-22",
    message="User input flows to open(). Validate file paths with os.path.realpath().",
    owasp="A01:2021",
)
def detect_django_path_traversal_open():
    """Detects Django request data flowing to open() file operations."""
    return flows(
        from_sources=_DJANGO_SOURCES,
        to_sinks=[
            IOModule.method("open").tracks(0),
            calls("open"),
        ],
        sanitized_by=[
            calls("os.path.realpath"),
            calls("os.path.abspath"),
            calls("os.path.basename"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )


@python_rule(
    id="PYTHON-DJANGO-SEC-041",
    name="Django Path Traversal via os.path.join()",
    severity="HIGH",
    category="django",
    cwe="CWE-22",
    tags="python,django,path-traversal,os-path,owasp-a01,cwe-22",
    message="User input flows to os.path.join() then to file operations. Validate paths.",
    owasp="A01:2021",
)
def detect_django_path_traversal_join():
    """Detects Django request data in os.path.join() reaching file operations."""
    return flows(
        from_sources=_DJANGO_SOURCES,
        to_sinks=[
            calls("open"),
            calls("os.path.join"),
        ],
        sanitized_by=[
            calls("os.path.realpath"),
            calls("os.path.abspath"),
            calls("os.path.basename"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
