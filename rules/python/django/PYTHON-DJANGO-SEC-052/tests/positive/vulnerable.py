from django.http import HttpResponse, HttpResponseBadRequest
from django.utils.safestring import mark_safe, SafeString
from django.utils.html import html_safe

# SEC-052: html_safe (audit)
@html_safe
class MyWidget:
    def __str__(self):
        return "<div>widget</div>"
