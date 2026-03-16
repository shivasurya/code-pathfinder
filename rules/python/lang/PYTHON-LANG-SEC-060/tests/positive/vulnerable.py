import requests as http_requests
import urllib.request
import ftplib
import telnetlib

# SEC-060: requests with HTTP
resp = http_requests.get("http://example.com/api")
http_requests.post("http://example.com/data", data={"key": "val"})
