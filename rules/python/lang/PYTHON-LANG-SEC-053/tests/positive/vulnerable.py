import ssl
import http.client
import requests as http_requests

resp = http_requests.get("https://example.com", verify=False)
