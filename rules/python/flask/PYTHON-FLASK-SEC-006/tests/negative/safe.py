from flask import Flask, request
from urllib.parse import urlparse
import requests
import ipaddress

ALLOWED_HOSTS = {'api.example.com', 'cdn.example.com'}

def validate_url(url):
    parsed = urlparse(url)
    if parsed.scheme not in ('http', 'https'):
        raise ValueError('Invalid scheme')
    if parsed.hostname not in ALLOWED_HOSTS:
        raise ValueError('Host not in allowlist')
    # Block private/reserved IP ranges
    try:
        ip = ipaddress.ip_address(parsed.hostname)
        if ip.is_private or ip.is_loopback or ip.is_link_local:
            raise ValueError('Private IP not allowed')
    except ValueError:
        pass  # hostname is not an IP, DNS will resolve it
    return url

app = Flask(__name__)

@app.route('/fetch')
def fetch_url():
    url = request.args.get('url')
    # SAFE: URL validated against allowlist before use
    safe_url = validate_url(url)
    response = requests.get(safe_url)
    return response.text
