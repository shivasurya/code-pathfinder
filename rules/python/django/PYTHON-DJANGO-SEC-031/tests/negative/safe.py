from django.http import JsonResponse
import requests
from urllib.parse import urlparse
import ipaddress

ALLOWED_DOMAINS = {'api.example.com', 'cdn.example.com'}
BLOCKED_NETWORKS = [
    ipaddress.ip_network('10.0.0.0/8'),
    ipaddress.ip_network('172.16.0.0/12'),
    ipaddress.ip_network('192.168.0.0/16'),
    ipaddress.ip_network('169.254.0.0/16'),  # Cloud metadata
    ipaddress.ip_network('127.0.0.0/8'),     # Loopback
]

def is_safe_url(url):
    \"\"\"Validate URL against allowlist and block internal networks.\"\"\"
    parsed = urlparse(url)
    if parsed.scheme not in ('http', 'https'):
        return False
    if parsed.hostname in ALLOWED_DOMAINS:
        return True
    try:
        ip = ipaddress.ip_address(parsed.hostname)
        return not any(ip in network for network in BLOCKED_NETWORKS)
    except ValueError:
        return False

def fetch_url(request):
    # SECURE: Validate URL before making request
    url = request.GET.get('url', '')
    if not is_safe_url(url):
        return JsonResponse({'error': 'URL not allowed'}, status=400)
    response = requests.get(url, timeout=5)
    return JsonResponse({'content': response.text})
