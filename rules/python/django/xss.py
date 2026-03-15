"""
Python Django XSS Rules

Rules:
- PYTHON-DJANGO-SEC-050: Direct HttpResponse Usage (CWE-79, Audit)
- PYTHON-DJANGO-SEC-051: mark_safe() Usage (CWE-79, Audit)
- PYTHON-DJANGO-SEC-052: html_safe() Usage (CWE-79, Audit)
- PYTHON-DJANGO-SEC-053: Class Extends SafeString (CWE-79, Audit)

Security Impact: MEDIUM to HIGH
CWE: CWE-79 (Improper Neutralization of Input During Web Page Generation)
OWASP: A03:2021 - Injection

DESCRIPTION:
These rules detect Cross-Site Scripting (XSS) vulnerabilities in Django applications.
Django provides automatic HTML escaping in templates by default, but developers can bypass
this protection by using HttpResponse directly with user input, calling mark_safe() on
untrusted data, using the html_safe() decorator, or extending SafeString. When user-controlled
input is rendered as unescaped HTML in the browser, attackers can inject malicious JavaScript
that executes in the context of the victim's session.

SECURITY IMPLICATIONS:

**1. Session Hijacking**:
Injected JavaScript can steal session cookies (document.cookie) and send them to an
attacker-controlled server, allowing the attacker to impersonate the victim and take
over their account.

**2. Credential Theft**:
XSS payloads can create fake login forms overlaying the legitimate page, capturing
credentials when users attempt to re-authenticate, or inject keyloggers to capture
all keystrokes on the page.

**3. Malware Distribution**:
Attackers can redirect users to malicious sites, trigger drive-by downloads, or inject
cryptocurrency mining scripts that run in the victim's browser.

**4. Defacement and Phishing**:
XSS enables attackers to modify page content, display misleading information, insert
phishing forms, or redirect users to look-alike sites to steal additional credentials.

VULNERABLE EXAMPLE:
```python
from django.http import HttpResponse, HttpResponseBadRequest
from django.utils.safestring import mark_safe, SafeString
from django.utils.html import html_safe


# SEC-050: HttpResponse with request data
def vulnerable_httpresponse(request):
    name = request.GET.get('name')
    return HttpResponse(f"Hello {name}")


def vulnerable_httpresponse_bad(request):
    msg = request.GET.get('error')
    return HttpResponseBadRequest(msg)


# SEC-051: mark_safe (audit)
def risky_mark_safe():
    content = "<script>alert(1)</script>"
    return mark_safe(content)


# SEC-052: html_safe (audit)
@html_safe
class MyWidget:
    def __str__(self):
        return "<div>widget</div>"


# SEC-053: SafeString subclass (audit)
custom = SafeString("<b>bold</b>")
```

SECURE EXAMPLE:
```python
from django.http import HttpResponse
from django.utils.html import escape, strip_tags
from django.template import loader

def greet_user(request):
    # SECURE: Use Django template with auto-escaping
    template = loader.get_template('greet.html')
    name = request.GET.get('name', '')
    return HttpResponse(template.render({'name': name}))
    # Django templates auto-escape {{ name }} by default

def greet_user_direct(request):
    # SECURE: Explicitly escape user input in HttpResponse
    name = escape(request.GET.get('name', ''))
    return HttpResponse("<h1>Hello, " + name + "!</h1>")

def render_comment(request):
    # SECURE: Sanitize before marking as safe, or use template auto-escaping
    comment = request.POST.get('comment', '')
    # Option 1: Strip all HTML tags
    clean_comment = strip_tags(comment)
    # Option 2: Use bleach to allow safe HTML subset
    # clean_comment = bleach.clean(comment, tags=['b', 'i', 'em', 'strong'])
    return render(request, 'comment.html', {'comment': clean_comment})
```

DETECTION AND PREVENTION:

**Key Mitigation Strategies**:
- Use Django templates with auto-escaping enabled (the default) instead of HttpResponse
- Never call mark_safe() on user-supplied or partially user-influenced content
- Use django.utils.html.escape() or conditional_escape() when building HTML strings
- Use strip_tags() or a library like bleach to sanitize HTML before rendering
- Set Content-Security-Policy headers to mitigate XSS impact
- Audit all mark_safe() and html_safe() usage in code reviews
- Use the |escape template filter explicitly for extra safety in templates

**Pre-deployment checks**:
```bash
pathfinder scan --project . --ruleset cpf/python/django/xss
```

COMPLIANCE:
- CWE-79: Improper Neutralization of Input During Web Page Generation
- OWASP A03:2021 - Injection
- SANS Top 25: CWE-79 ranked #2
- NIST SP 800-53: SI-10 (Information Input Validation)

REFERENCES:
- CWE-79: https://cwe.mitre.org/data/definitions/79.html
- OWASP XSS: https://owasp.org/www-community/attacks/xss/
- OWASP XSS Prevention Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Cross_Site_Scripting_Prevention_Cheat_Sheet.html
- Django auto-escaping: https://docs.djangoproject.com/en/stable/ref/templates/language/#automatic-html-escaping
- Django Security: https://docs.djangoproject.com/en/stable/topics/security/#cross-site-scripting-xss-protection
"""

from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets


class DjangoSafeString(QueryType):
    fqns = ["django.utils.safestring", "django.utils.html"]


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
    id="PYTHON-DJANGO-SEC-050",
    name="Django Direct HttpResponse Usage",
    severity="MEDIUM",
    category="django",
    cwe="CWE-79",
    tags="python,django,xss,httpresponse,owasp-a03,cwe-79",
    message="Direct HttpResponse with user input detected. Use templates with auto-escaping.",
    owasp="A03:2021",
)
def detect_django_httpresponse_xss():
    """Detects user input flowing to HttpResponse without escaping."""
    return flows(
        from_sources=_DJANGO_SOURCES,
        to_sinks=[
            calls("HttpResponse"),
            calls("HttpResponseBadRequest"),
            calls("HttpResponseNotFound"),
            calls("HttpResponseForbidden"),
            calls("HttpResponseServerError"),
        ],
        sanitized_by=[
            calls("escape"),
            calls("django.utils.html.escape"),
            calls("conditional_escape"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )


@python_rule(
    id="PYTHON-DJANGO-SEC-051",
    name="Django mark_safe() Usage (Audit)",
    severity="MEDIUM",
    category="django",
    cwe="CWE-79",
    tags="python,django,xss,mark-safe,audit,cwe-79",
    message="mark_safe() bypasses Django auto-escaping. Ensure input is properly sanitized.",
    owasp="A03:2021",
)
def detect_django_mark_safe():
    """Audit: detects mark_safe() usage that bypasses auto-escaping."""
    return DjangoSafeString.method("mark_safe")


@python_rule(
    id="PYTHON-DJANGO-SEC-052",
    name="Django html_safe() Usage (Audit)",
    severity="MEDIUM",
    category="django",
    cwe="CWE-79",
    tags="python,django,xss,html-safe,audit,cwe-79",
    message="html_safe() marks output as safe HTML. Ensure content is properly sanitized.",
    owasp="A03:2021",
)
def detect_django_html_safe():
    """Audit: detects html_safe() usage."""
    return DjangoSafeString.method("html_safe")


@python_rule(
    id="PYTHON-DJANGO-SEC-053",
    name="Django SafeString Subclass (Audit)",
    severity="MEDIUM",
    category="django",
    cwe="CWE-79",
    tags="python,django,xss,safestring,audit,cwe-79",
    message="Class extends SafeString/SafeData. Content bypasses auto-escaping.",
    owasp="A03:2021",
)
def detect_django_safestring_subclass():
    """Audit: detects SafeString/SafeText subclass usage."""
    return DjangoSafeString.method("SafeString")
