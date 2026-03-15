"""
Python Django Email Injection / XSS Rules

Rules:
- PYTHON-DJANGO-SEC-060: XSS in HTML Email Body (CWE-79)
- PYTHON-DJANGO-SEC-061: XSS in send_mail html_message (CWE-79)

Security Impact: MEDIUM
CWE: CWE-79 (Improper Neutralization of Input During Web Page Generation)
OWASP: A03:2021 - Injection

DESCRIPTION:
These rules detect Cross-Site Scripting (XSS) vulnerabilities in Django email sending
functions where untrusted user input flows into HTML email bodies without sanitization.
When user-controlled data is included in HTML email content via EmailMessage or send_mail()
with the html_message parameter, attackers can inject malicious HTML and JavaScript that
executes when the recipient opens the email in a client that renders HTML.

Unlike browser-based XSS, email XSS targets email clients, which have varying levels of
HTML/JavaScript support. While modern webmail clients (Gmail, Outlook.com) strip scripts,
some desktop email clients and older webmail systems may execute injected JavaScript. Even
without script execution, HTML injection in emails can be used for phishing attacks by
rendering convincing fake content within legitimate emails.

SECURITY IMPLICATIONS:

**1. Phishing via HTML Injection**:
Attackers can inject HTML content into legitimate emails to display fake login forms,
fraudulent payment requests, or misleading messages. Because the email originates from
a trusted domain, recipients are more likely to trust the content.

**2. Email Client XSS**:
In email clients that render JavaScript, injected scripts can access other emails,
steal contact lists, or trigger actions within the email client on behalf of the victim.

**3. Content Spoofing**:
Injected HTML can completely replace the visible email content, overriding the sender's
intended message with attacker-controlled content while maintaining the legitimate sender
address and headers.

**4. Link Manipulation**:
Attackers can inject links that appear to point to legitimate sites but redirect to
phishing pages, malware downloads, or credential-harvesting forms.

VULNERABLE EXAMPLE:
```python
from django.core.mail import EmailMessage, send_mail


# SEC-060: XSS in email body
def vulnerable_email(request):
    body = request.POST.get('message')
    email = EmailMessage("Subject", body, "from@test.com", ["to@test.com"])
    email.content_subtype = "html"
    email.send()


# SEC-061: XSS in send_mail html_message
def vulnerable_sendmail(request):
    content = request.POST.get('body')
    send_mail("Subject", "text body", "from@test.com", ["to@test.com"],
              html_message=content)
```

SECURE EXAMPLE:
```python
from django.core.mail import send_mail
from django.utils.html import escape, strip_tags
from django.template.loader import render_to_string

def send_welcome(request):
    # SECURE: Escape user input before including in HTML email
    name = escape(request.POST.get('name', ''))
    html_body = render_to_string('emails/welcome.html', {'name': name})
    send_mail(
        'Welcome!',
        strip_tags(html_body),  # Plain text fallback
        'noreply@example.com',
        [request.POST.get('email')],
        html_message=html_body,
    )

def send_notification(request):
    # SECURE: Use Django template for HTML email with auto-escaping
    message = request.POST.get('message', '')
    context = {'message': message}  # Auto-escaped in template
    html_content = render_to_string('emails/notification.html', context)
    send_mail(
        'Notification',
        strip_tags(html_content),
        'noreply@example.com',
        ['admin@example.com'],
        html_message=html_content,
    )
```

DETECTION AND PREVENTION:

**Key Mitigation Strategies**:
- Use Django templates with auto-escaping for HTML email content
- Apply django.utils.html.escape() to all user input before embedding in HTML emails
- Use strip_tags() to remove all HTML from user input when plain text is sufficient
- Provide both plain text and HTML versions of emails (multipart/alternative)
- Use Content-Security-Policy in HTML emails where supported
- Validate and sanitize email addresses to prevent header injection
- Use a library like bleach to allow only a safe subset of HTML tags

**Pre-deployment checks**:
```bash
pathfinder scan --project . --ruleset cpf/python/django/email-injection
```

COMPLIANCE:
- CWE-79: Improper Neutralization of Input During Web Page Generation
- OWASP A03:2021 - Injection
- SANS Top 25: CWE-79 ranked #2
- NIST SP 800-53: SI-10 (Information Input Validation)

REFERENCES:
- CWE-79: https://cwe.mitre.org/data/definitions/79.html
- OWASP XSS: https://owasp.org/www-community/attacks/xss/
- Django send_mail: https://docs.djangoproject.com/en/stable/topics/email/#send-mail
- Django EmailMessage: https://docs.djangoproject.com/en/stable/topics/email/#emailmessage-objects
- Django Security: https://docs.djangoproject.com/en/stable/topics/security/
"""

from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets


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
    id="PYTHON-DJANGO-SEC-060",
    name="Django XSS in HTML Email Body",
    severity="MEDIUM",
    category="django",
    cwe="CWE-79",
    tags="python,django,xss,email,html,owasp-a03,cwe-79",
    message="User input in HTML email body. Sanitize content before sending.",
    owasp="A03:2021",
)
def detect_django_email_xss():
    """Detects user input flowing into EmailMessage body."""
    return flows(
        from_sources=_DJANGO_SOURCES,
        to_sinks=[
            calls("EmailMessage"),
            calls("django.core.mail.EmailMessage"),
        ],
        sanitized_by=[
            calls("escape"),
            calls("strip_tags"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )


@python_rule(
    id="PYTHON-DJANGO-SEC-061",
    name="Django XSS in send_mail html_message",
    severity="MEDIUM",
    category="django",
    cwe="CWE-79",
    tags="python,django,xss,send-mail,html,owasp-a03,cwe-79",
    message="User input in send_mail() html_message parameter. Sanitize content.",
    owasp="A03:2021",
)
def detect_django_sendmail_xss():
    """Detects user input flowing into send_mail() html_message."""
    return flows(
        from_sources=_DJANGO_SOURCES,
        to_sinks=[
            calls("send_mail"),
            calls("django.core.mail.send_mail"),
        ],
        sanitized_by=[
            calls("escape"),
            calls("strip_tags"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
