import requests as http_requests
import urllib.request
import ftplib
import telnetlib

# SEC-064: telnetlib
tn = telnetlib.Telnet("example.com", 23)
