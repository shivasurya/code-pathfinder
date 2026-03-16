import requests as http_requests
import urllib.request
import ftplib
import telnetlib

# SEC-063: FTP without TLS
ftp = ftplib.FTP("ftp.example.com")
