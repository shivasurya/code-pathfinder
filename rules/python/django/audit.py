"""
Python Django Security Audit Rules

Rules:
- PYTHON-DJANGO-SEC-070: Insecure Cookie Settings (CWE-614)
- PYTHON-DJANGO-SEC-071: CSRF Exempt Decorator (CWE-352)
- PYTHON-DJANGO-SEC-072: Insecure Deserialization of Request Data (CWE-502)

Security Impact: MEDIUM to CRITICAL
CWE: CWE-502 (Deserialization of Untrusted Data),
     CWE-352 (Cross-Site Request Forgery),
     CWE-614 (Sensitive Cookie in HTTPS Session Without 'Secure' Attribute)
OWASP: A05:2021 - Security Misconfiguration, A08:2021 - Software and Data Integrity Failures

DESCRIPTION:
These rules detect common security misconfigurations and unsafe patterns in Django
applications. They cover three distinct vulnerability categories:

1. **Insecure Cookie Settings (CWE-614)**: Detects set_cookie() calls that may lack the
   secure, httponly, or samesite flags. Without these flags, cookies can be intercepted
   over unencrypted connections, accessed by JavaScript (enabling XSS-based session theft),
   or sent in cross-site requests (enabling CSRF attacks).

2. **CSRF Exempt Decorator (CWE-352)**: Detects usage of Django's @csrf_exempt decorator,
   which disables Cross-Site Request Forgery protection on views. This allows attackers to
   craft malicious pages that trigger state-changing actions on behalf of authenticated
   users without their consent.

3. **Insecure Deserialization (CWE-502)**: Detects request data flowing into unsafe
   deserialization functions (pickle, yaml.load, dill, shelve). These functions can execute
   arbitrary code during deserialization, making them critical attack vectors when used with
   untrusted input.

SECURITY IMPLICATIONS:

**1. Session Hijacking via Insecure Cookies**:
Without the Secure flag, session cookies are transmitted over HTTP in plaintext, allowing
network attackers to intercept them via man-in-the-middle attacks. Without HttpOnly,
JavaScript can read cookies, enabling XSS-to-session-theft attacks.

**2. Cross-Site Request Forgery**:
@csrf_exempt views are vulnerable to CSRF attacks where a malicious website tricks an
authenticated user's browser into making requests (transfers, password changes, data
deletion) to the Django application without the user's knowledge.

**3. Remote Code Execution via Deserialization**:
When request data flows to pickle.loads(), yaml.load(), or similar functions, attackers
can craft payloads that execute arbitrary code on the server. This is one of the most
severe vulnerability classes, often leading to complete server compromise.

VULNERABLE EXAMPLE:
```python
from django.http import HttpResponse
from django.views.decorators.csrf import csrf_exempt
import pickle
import yaml

def set_preferences(request):
    # VULNERABLE: Cookie set without security flags
    response = HttpResponse("Preferences saved")
    response.set_cookie('session_id', request.session.session_key)
    # Missing: secure=True, httponly=True, samesite='Lax'
    return response

@csrf_exempt  # VULNERABLE: CSRF protection disabled
def transfer_funds(request):
    amount = request.POST.get('amount')
    recipient = request.POST.get('recipient')
    # Attacker can trigger this from any website
    perform_transfer(request.user, recipient, amount)
    return HttpResponse("Transfer complete")

@csrf_exempt
def load_data(request):
    # VULNERABLE: Request body deserialized with pickle
    data = pickle.loads(request.body)  # RCE!
    return HttpResponse(str(data))
```

SECURE EXAMPLE:
```python
from django.http import HttpResponse, JsonResponse
from django.middleware.csrf import CsrfViewMiddleware
import json

def set_preferences(request):
    # SECURE: Cookie set with all security flags
    response = HttpResponse("Preferences saved")
    response.set_cookie(
        'session_id',
        request.session.session_key,
        secure=True,      # Only sent over HTTPS
        httponly=True,     # Not accessible via JavaScript
        samesite='Lax',   # Prevents cross-site sending
        max_age=3600,     # 1 hour expiry
    )
    return response

# SECURE: No @csrf_exempt - CSRF protection enabled by default
def transfer_funds(request):
    if request.method != 'POST':
        return HttpResponse(status=405)
    amount = request.POST.get('amount')
    recipient = request.POST.get('recipient')
    perform_transfer(request.user, recipient, amount)
    return HttpResponse("Transfer complete")

def load_data(request):
    # SECURE: Use JSON instead of pickle for untrusted data
    try:
        data = json.loads(request.body)
    except json.JSONDecodeError:
        return JsonResponse({'error': 'Invalid JSON'}, status=400)
    return JsonResponse({'data': data})
```

DETECTION AND PREVENTION:

**Key Mitigation Strategies**:
- Always set secure=True, httponly=True, and samesite='Lax' (or 'Strict') on cookies
- Configure SESSION_COOKIE_SECURE, SESSION_COOKIE_HTTPONLY, and CSRF_COOKIE_SECURE in Django settings
- Avoid @csrf_exempt; use it only for genuinely public API endpoints with alternative auth
- For API endpoints needing CSRF exemption, use token-based authentication (JWT, API keys)
- Never use pickle, yaml.load(), or dill with untrusted data; use json.loads() instead
- Use yaml.safe_load() instead of yaml.load() when YAML parsing is required
- Audit all @csrf_exempt decorators during security reviews

**Pre-deployment checks**:
```bash
pathfinder scan --project . --ruleset cpf/python/django/audit
```

COMPLIANCE:
- CWE-502: Deserialization of Untrusted Data
- CWE-352: Cross-Site Request Forgery (CSRF)
- CWE-614: Sensitive Cookie in HTTPS Session Without 'Secure' Attribute
- OWASP A05:2021 - Security Misconfiguration
- OWASP A08:2021 - Software and Data Integrity Failures
- SANS Top 25: Insecure Deserialization, CSRF
- NIST SP 800-53: SC-23 (Session Authenticity), SI-10 (Information Input Validation)

REFERENCES:
- CWE-502: https://cwe.mitre.org/data/definitions/502.html
- CWE-352: https://cwe.mitre.org/data/definitions/352.html
- CWE-614: https://cwe.mitre.org/data/definitions/614.html
- OWASP CSRF: https://owasp.org/www-community/attacks/csrf
- Django CSRF Protection: https://docs.djangoproject.com/en/stable/ref/csrf/
- Django Cookie Security: https://docs.djangoproject.com/en/stable/ref/settings/#session-cookie-secure
- Django Security: https://docs.djangoproject.com/en/stable/topics/security/
"""

from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets


class PickleModule(QueryType):
    fqns = ["pickle", "_pickle", "cPickle"]


class YamlModule(QueryType):
    fqns = ["yaml"]


class DillModule(QueryType):
    fqns = ["dill"]


class ShelveModule(QueryType):
    fqns = ["shelve"]


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
    id="PYTHON-DJANGO-SEC-070",
    name="Django Insecure Cookie Settings",
    severity="MEDIUM",
    category="django",
    cwe="CWE-614",
    tags="python,django,cookies,security,owasp-a05,cwe-614",
    message="set_cookie() without secure/httponly flags. Set secure=True, httponly=True, samesite='Lax'.",
    owasp="A05:2021",
)
def detect_django_insecure_cookies():
    """Audit: detects set_cookie() calls that may lack security attributes."""
    return calls("*.set_cookie")


@python_rule(
    id="PYTHON-DJANGO-SEC-071",
    name="Django CSRF Exempt Decorator",
    severity="MEDIUM",
    category="django",
    cwe="CWE-352",
    tags="python,django,csrf,security,owasp-a05,cwe-352",
    message="@csrf_exempt disables CSRF protection. Ensure this is intentional.",
    owasp="A05:2021",
)
def detect_django_csrf_exempt():
    """Audit: detects @csrf_exempt decorator usage."""
    return calls("csrf_exempt")


@python_rule(
    id="PYTHON-DJANGO-SEC-072",
    name="Django Insecure Deserialization of Request Data",
    severity="CRITICAL",
    category="django",
    cwe="CWE-502",
    tags="python,django,deserialization,pickle,yaml,owasp-a08,cwe-502",
    message="Request data flows to unsafe deserialization. Use JSON instead of pickle/yaml.",
    owasp="A08:2021",
)
def detect_django_insecure_deserialization():
    """Detects request data flowing to pickle/yaml/dill/shelve deserialization."""
    return flows(
        from_sources=_DJANGO_SOURCES,
        to_sinks=[
            PickleModule.method("loads", "load"),
            YamlModule.method("load", "unsafe_load"),
            DillModule.method("loads", "load"),
            calls("shelve.open"),
        ],
        sanitized_by=[],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
