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
