from django.http import HttpResponse, HttpResponseBadRequest
from django.utils.safestring import mark_safe, SafeString
from django.utils.html import html_safe

# SEC-053: SafeString subclass (audit)
custom = SafeString("<b>bold</b>")
