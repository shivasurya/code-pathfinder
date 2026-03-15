"""
Python Django Password Security Rules

Rules:
- PYTHON-DJANGO-SEC-080: Empty Password in set_password() (CWE-521)
- PYTHON-DJANGO-SEC-081: Default Empty Password Value (CWE-521)

Security Impact: HIGH
CWE: CWE-521 (Weak Password Requirements)
OWASP: A07:2021 - Identification and Authentication Failures

DESCRIPTION:
These rules detect weak password handling patterns in Django applications where passwords
may be set to empty strings or where request.POST.get() with a default empty string value
flows into set_password(). Django's set_password() with an empty string creates a user
account with no password protection, effectively disabling authentication for that account.
The correct approach is to use None (not empty string) when no password should be set,
which makes the password unusable via Django's set_unusable_password() mechanism.

SECURITY IMPLICATIONS:

**1. Account Takeover**:
If set_password('') is called, the account has an empty password. Depending on the
authentication backend, this may allow login with a blank password or bypass
authentication entirely, giving attackers full access to the account.

**2. Mass Account Compromise**:
When request.POST.get('password', '') is used and the password field is missing from the
form submission (e.g., due to a frontend bug or manipulated request), the default empty
string silently sets an empty password, potentially compromising many accounts.

**3. Privilege Escalation**:
If administrative or privileged accounts have their passwords inadvertently set to empty
strings, attackers can escalate privileges by logging in as those accounts.

**4. Compliance Violations**:
Empty passwords violate password policies required by PCI DSS, HIPAA, SOC 2, and other
compliance frameworks, which mandate minimum password complexity and length requirements.

VULNERABLE EXAMPLE:
```python
from django.contrib.auth.models import User

def reset_password(request):
    # VULNERABLE: POST.get defaults to empty string if 'password' is missing
    password = request.POST.get('password')  # Returns '' if missing
    user = User.objects.get(id=request.user.id)
    user.set_password(password)  # Empty string = no password!
    user.save()

def create_user(request):
    # VULNERABLE: Explicitly setting empty password
    user = User.objects.create_user(
        username=request.POST.get('username'),
        password='',  # Empty password - account has no protection
    )
```

SECURE EXAMPLE:
```python
from django.contrib.auth.models import User
from django.core.exceptions import ValidationError

def reset_password(request):
    # SECURE: Validate password is present and meets requirements
    password = request.POST.get('password')
    if not password or len(password) < 8:
        raise ValidationError("Password must be at least 8 characters")
    user = User.objects.get(id=request.user.id)
    user.set_password(password)
    user.save()

def create_user_no_password(request):
    # SECURE: Use set_unusable_password() instead of empty string
    user = User.objects.create_user(
        username=request.POST.get('username'),
    )
    user.set_unusable_password()  # Correct way to create account without password
    user.save()

def create_user_with_validation(request):
    # SECURE: Use Django's password validators
    from django.contrib.auth.password_validation import validate_password
    password = request.POST.get('password')
    validate_password(password)  # Raises ValidationError if too weak
    user = User.objects.create_user(
        username=request.POST.get('username'),
        password=password,
    )
```

DETECTION AND PREVENTION:

**Key Mitigation Strategies**:
- Never pass empty strings to set_password(); use set_unusable_password() instead
- Always validate that password fields are non-empty before calling set_password()
- Use Django's built-in password validators (AUTH_PASSWORD_VALIDATORS in settings.py)
- Require minimum password length (8+ characters) and complexity
- Use request.POST.get('password') with explicit None-check, not default empty string
- Implement rate limiting on password reset endpoints
- Log and monitor password change events for anomalies

**Pre-deployment checks**:
```bash
pathfinder scan --project . --ruleset cpf/python/django/passwords
```

COMPLIANCE:
- CWE-521: Weak Password Requirements
- OWASP A07:2021 - Identification and Authentication Failures
- NIST SP 800-63B: Digital Identity Guidelines (Authentication)
- PCI DSS Requirement 8.2: Proper password management
- SANS Top 25: Authentication failures

REFERENCES:
- CWE-521: https://cwe.mitre.org/data/definitions/521.html
- OWASP Authentication Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html
- Django Password Management: https://docs.djangoproject.com/en/stable/topics/auth/passwords/
- Django Password Validation: https://docs.djangoproject.com/en/stable/ref/settings/#auth-password-validators
"""

from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets


@python_rule(
    id="PYTHON-DJANGO-SEC-080",
    name="Django Empty Password in set_password()",
    severity="HIGH",
    category="django",
    cwe="CWE-521",
    tags="python,django,password,empty,owasp-a07,cwe-521",
    message="Empty password set via set_password(). Use None instead of empty string.",
    owasp="A07:2021",
)
def detect_django_empty_password():
    """Audit: detects set_password() calls that may use empty strings."""
    return calls("*.set_password")


@python_rule(
    id="PYTHON-DJANGO-SEC-081",
    name="Django Default Empty Password Value",
    severity="HIGH",
    category="django",
    cwe="CWE-521",
    tags="python,django,password,default,owasp-a07,cwe-521",
    message="Password default value may be empty string. Use None as default.",
    owasp="A07:2021",
)
def detect_django_default_empty_password():
    """Audit: detects request.POST.get with potential empty password default flowing to set_password."""
    return flows(
        from_sources=[
            calls("request.POST.get"),
            calls("*.POST.get"),
        ],
        to_sinks=[
            calls("*.set_password"),
        ],
        sanitized_by=[],
        propagates_through=PropagationPresets.standard(),
        scope="local",
    )
