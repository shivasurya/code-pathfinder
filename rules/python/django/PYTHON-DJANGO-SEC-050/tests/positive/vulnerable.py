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
