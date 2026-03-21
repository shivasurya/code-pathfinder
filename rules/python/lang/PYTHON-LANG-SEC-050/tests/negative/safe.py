import ssl, requests
# Use default context with proper certificate verification
ctx = ssl.create_default_context()
response = requests.get("https://api.example.com")  # verify=True is default
# Enforce minimum TLS 1.2
ctx = ssl.SSLContext(ssl.PROTOCOL_TLS_CLIENT)
ctx.minimum_version = ssl.TLSVersion.TLSv1_2
# Use SSLContext.wrap_socket with hostname checking
ctx = ssl.create_default_context()
secure_sock = ctx.wrap_socket(sock, server_hostname="api.example.com")
# Always use HTTPS
conn = http.client.HTTPSConnection("api.example.com")
