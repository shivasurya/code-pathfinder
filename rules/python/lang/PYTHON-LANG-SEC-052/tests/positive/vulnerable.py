import ssl
import http.client
import requests as http_requests

# SEC-052: deprecated wrap_socket
wrapped = ssl.wrap_socket(sock)
