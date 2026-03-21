from django.http import HttpResponse, HttpResponseBadRequest
from django.utils.safestring import mark_safe, SafeString
from django.utils.html import html_safe

# SEC-051: mark_safe (audit)
def risky_mark_safe():
    content = "<script>alert(1)</script>"
    return mark_safe(content)
