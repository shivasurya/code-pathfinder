import requests

def fetch_url(request):
    url = request.GET.get('url')
    resp = requests.get(url)
    return resp.text
