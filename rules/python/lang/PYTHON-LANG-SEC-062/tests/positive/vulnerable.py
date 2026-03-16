import requests as http_requests
import urllib.request
import ftplib
import telnetlib

# SEC-060: requests with HTTP
resp = http_requests.get("http://example.com/api")
http_requests.post("http://example.com/data", data={"key": "val"})

# SEC-061: urllib insecure
urllib.request.urlopen("http://example.com")
urllib.request.urlretrieve("http://example.com/file", "local.txt")

# SEC-062: urllib Request
req = urllib.request.Request("http://example.com")

# SEC-063: FTP without TLS
ftp = ftplib.FTP("ftp.example.com")

# SEC-064: telnetlib
tn = telnetlib.Telnet("example.com", 23)
