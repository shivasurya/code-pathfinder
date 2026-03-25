from flask import request, abort

# SEC-113: Host header used for access control decisions

# 1. Host header check in authentication middleware
def check_admin_access():
    host = request.environ.get("HTTP_HOST")
    if host == "admin.internal.company.com":
        return True
    abort(403)

# 2. Host header used to determine allowed origins
def validate_origin():
    host = request.environ.get("HTTP_HOST")
    allowed_hosts = ["api.example.com", "app.example.com"]
    if host not in allowed_hosts:
        abort(403)
    return True

# 3. Host header in URL construction for redirect
def build_redirect_url(path):
    host = request.environ.get("HTTP_HOST")
    return f"https://{host}/{path}"

# 4. Host header used for tenant isolation
def get_tenant_config():
    host = request.environ.get("HTTP_HOST")
    tenant = host.split(".")[0]
    return load_tenant_config(tenant)

# 5. Host header passed to downstream service
def proxy_request():
    host = request.environ.get("HTTP_HOST")
    headers = {"X-Forwarded-Host": host}
    return forward_request(headers)

# 6. Host header in WSGI app
def wsgi_app(environ, start_response):
    host = environ.get("HTTP_HOST")
    if host != "trusted.example.com":
        start_response("403 Forbidden", [])
        return [b"Forbidden"]
    start_response("200 OK", [])
    return [b"OK"]
