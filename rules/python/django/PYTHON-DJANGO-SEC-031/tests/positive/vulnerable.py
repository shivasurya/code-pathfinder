import requests
import urllib.request


# SEC-030: SSRF via requests
def vulnerable_ssrf_requests(request):
    url = request.GET.get('url')
    resp = requests.get(url)
    return resp.text


# SEC-031: SSRF via urllib
def vulnerable_ssrf_urllib(request):
    url = request.POST.get('target')
    resp = urllib.request.urlopen(url)
    return resp.read()
