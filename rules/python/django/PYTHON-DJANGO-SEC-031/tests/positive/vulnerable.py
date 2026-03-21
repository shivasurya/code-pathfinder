from urllib.request import urlopen

def fetch_url(request):
    url = request.GET.get('url')
    resp = urlopen(url)
    return resp.read()
