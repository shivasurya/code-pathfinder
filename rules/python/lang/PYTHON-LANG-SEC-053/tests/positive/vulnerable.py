import ssl
import http.client
import requests as http_requests

# SEC-050: unverified SSL context
ctx = ssl._create_unverified_context()

# SEC-051: weak SSL version
ctx2 = ssl.SSLContext(ssl.PROTOCOL_SSLv2)

# SEC-052: deprecated wrap_socket
wrapped = ssl.wrap_socket(sock)

# SEC-053: disabled cert validation
resp = http_requests.get("https://example.com", verify=False)

# SEC-054: HTTP instead of HTTPS
conn = http.client.HTTPConnection("example.com")
