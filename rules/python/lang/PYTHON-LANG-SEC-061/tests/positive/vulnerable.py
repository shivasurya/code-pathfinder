import requests as http_requests
import urllib.request
import ftplib
import telnetlib

# SEC-061: urllib insecure
urllib.request.urlopen("http://example.com")
urllib.request.urlretrieve("http://example.com/file", "local.txt")
