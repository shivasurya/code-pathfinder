import requests, ftplib
# Always use HTTPS for network requests
response = requests.get("https://api.example.com/users")
# Use FTP over TLS (FTPS)
ftp = ftplib.FTP_TLS("ftp.example.com")
ftp.auth()  # Establish TLS
ftp.login("user", "password")  # Encrypted
# Use SSH (paramiko) instead of telnet
import paramiko
ssh = paramiko.SSHClient()
ssh.connect("server.example.com", username="user", key_filename="key.pem")
