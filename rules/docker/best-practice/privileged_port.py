"""
DOCKER-AUD-003: Privileged Port Exposed

Security Impact: MEDIUM
Category: Audit / Security

DESCRIPTION:
This rule detects EXPOSE instructions for ports below 1024 (privileged ports).
On Unix-like systems, binding to ports 1-1023 traditionally requires root privileges,
which conflicts with the security best practice of running containers as non-root users.
While this rule is informational (sometimes privileged ports are intentional), it
highlights a potential privilege escalation requirement.

UNIX PRIVILEGED PORTS BACKGROUND:

Ports 1-1023 are historically "privileged" or "well-known" ports that require
root (UID 0) to bind on Unix/Linux systems. This restriction was designed to
prevent unprivileged users from impersonating system services.

Common privileged ports:
- 20/21: FTP
- 22: SSH
- 23: Telnet
- 25: SMTP
- 53: DNS
- 80: HTTP
- 443: HTTPS
- 110: POP3
- 143: IMAP
- 389: LDAP
- 445: SMB
- 3306: MySQL (exception: not privileged, but commonly exposed)

SECURITY IMPLICATIONS:

**1. Requires Root to Bind**:
```dockerfile
FROM nginx:alpine

# Exposes privileged port 80
EXPOSE 80

# nginx master process must run as root to bind to port 80
# This violates least privilege principle
```

**2. Container Must Run as Root**:
If you expose port 80 and want to bind directly:
```bash
$ docker run -p 80:80 --user 1000 nginx
# Error: nginx: [emerg] bind() to 0.0.0.0:80 failed (13: Permission denied)
```

You're forced to run as root:
```bash
$ docker run -p 80:80 nginx
# Works, but container runs as root (security risk)
```

**3. Attack Surface Expansion**:
- Root processes have more capabilities (CAP_NET_BIND_SERVICE)
- If app is compromised, attacker has root inside container
- Easier to escalate to host if combined with other vulnerabilities

VULNERABLE EXAMPLE:
```dockerfile
FROM ubuntu:22.04

RUN apt-get update && \
    apt-get install -y --no-install-recommends nginx && \
    rm -rf /var/lib/apt/lists/*

# Bad: Requires root to bind
EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
```

SECURE EXAMPLE:

**Solution 1: Use Non-Privileged Ports**
```dockerfile
FROM nginx:alpine

# Use port 8080 instead of 80 (no root needed)
EXPOSE 8080

# Configure nginx to listen on 8080
COPY nginx.conf /etc/nginx/nginx.conf

# Run as non-root user
USER nginx

CMD ["nginx", "-g", "daemon off;"]
```

nginx.conf:
```nginx
server {
    listen 8080;
    server_name _;
    ...
}
```

Run with port mapping:
```bash
docker run -p 80:8080 myapp
# Host port 80 → Container port 8080
# Container process doesn't need root
```

**Solution 2: Use CAP_NET_BIND_SERVICE Capability**
```dockerfile
FROM ubuntu:22.04

RUN apt-get update && \
    apt-get install -y --no-install-recommends nginx libcap2-bin && \
    rm -rf /var/lib/apt/lists/*

# Grant nginx binary the capability to bind privileged ports
RUN setcap 'cap_net_bind_service=+ep' /usr/sbin/nginx

# Create non-root user
RUN useradd -r -u 999 nginx

EXPOSE 80

USER nginx

CMD ["nginx", "-g", "daemon off;"]
```

This grants only the specific capability needed, not full root:
```bash
# Check capabilities
docker run --rm myapp getcap /usr/sbin/nginx
# /usr/sbin/nginx = cap_net_bind_service+ep
```

**Solution 3: Use Reverse Proxy/Load Balancer**
```yaml
# docker-compose.yml
version: '3.8'
services:
  # Reverse proxy handles privileged ports
  proxy:
    image: traefik:v2.10
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro

  # App runs on non-privileged port
  app:
    build: .
    expose:
      - "8080"  # Not privileged
    labels:
      - "traefik.http.routers.app.rule=Host(`example.com`)"
      - "traefik.http.services.app.loadbalancer.server.port=8080"
```

**Solution 4: Systemd Socket Activation (Host-Level)**
```dockerfile
# App listens on port passed via systemd socket
FROM myapp:latest
EXPOSE 8080  # Non-privileged fallback
```

Host systemd unit:
```ini
[Socket]
ListenStream=80
[Install]
WantedBy=sockets.target
```

**Solution 5: Kubernetes Service Abstraction**
```dockerfile
# Container exposes high port
FROM myapp:latest
EXPOSE 8080
```

```yaml
# Kubernetes Service handles external port 80
apiVersion: v1
kind: Service
metadata:
  name: myapp
spec:
  ports:
  - port: 80        # External
    targetPort: 8080 # Container
  selector:
    app: myapp
```

LINUX CAPABILITIES APPROACH:

Instead of running as root, grant specific capability:

```dockerfile
FROM ubuntu:22.04

# Install setcap utility
RUN apt-get update && \
    apt-get install -y --no-install-recommends libcap2-bin

# Install your application
COPY myapp /usr/local/bin/myapp

# Grant capability to bind privileged ports
RUN setcap cap_net_bind_service=+ep /usr/local/bin/myapp

# Create non-root user
RUN useradd -r -u 999 appuser

EXPOSE 80

USER appuser

CMD ["/usr/local/bin/myapp"]
```

Run container:
```bash
docker run -p 80:80 myapp
# Works! Container runs as UID 999, not root
```

COMMON PORT REMAPPING STRATEGIES:

```
Privileged Port → Non-Privileged Port
-----------------------------------------
80  (HTTP)      → 8080, 8000, 3000
443 (HTTPS)     → 8443, 4443
22  (SSH)       → 2222
25  (SMTP)      → 2525
53  (DNS)       → 5353
389 (LDAP)      → 3389
```

Example Dockerfile with remapping:
```dockerfile
FROM python:3.11-slim

WORKDIR /app

# Install dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy app
COPY . .

# Use non-privileged port
EXPOSE 8000

# Run as non-root
RUN useradd -r -u 999 appuser
USER appuser

CMD ["uvicorn", "app:app", "--host", "0.0.0.0", "--port", "8000"]
```

Deployment with port mapping:
```bash
# Development
docker run -p 8000:8000 myapp

# Production (map to standard HTTP port)
docker run -p 80:8000 myapp
```

FRAMEWORK-SPECIFIC CONFIGURATIONS:

**Node.js/Express**:
```javascript
// app.js - Use environment variable for port
const PORT = process.env.PORT || 8080;
app.listen(PORT, () => {
  console.log(`Server running on port ${PORT}`);
});
```

**Flask/Python**:
```python
# app.py
if __name__ == '__main__':
    port = int(os.environ.get('PORT', 8080))
    app.run(host='0.0.0.0', port=port)
```

**Go**:
```go
// main.go
port := os.Getenv("PORT")
if port == "" {
    port = "8080"
}
http.ListenAndServe(":"+port, handler)
```

**Spring Boot/Java**:
```properties
# application.properties
server.port=${PORT:8080}
```

WHEN PRIVILEGED PORTS ARE ACCEPTABLE:

1. **Legacy Systems** requiring specific ports (e.g., LDAP on 389)
2. **Protocol Requirements** (e.g., DNS on 53)
3. **Compliance Requirements** (e.g., HTTP must be exactly port 80)
4. **Testing/Development** environments (not production)

In these cases, document why privileged port is necessary and use CAP_NET_BIND_SERVICE.

VERIFICATION:

**Check what ports container is using**:
```bash
docker ps --format "table {{.Names}}\t{{.Ports}}"
```

**Verify container user**:
```bash
docker inspect myapp | jq '.[0].Config.User'
# Should not be empty or "root"
```

**Test port binding**:
```bash
docker run --rm myapp netstat -tuln | grep LISTEN
# Should show process listening on non-privileged port
```

REMEDIATION:

**Step 1: Change app to listen on port >= 1024**
```python
# Before
app.run(host='0.0.0.0', port=80)

# After
app.run(host='0.0.0.0', port=8080)
```

**Step 2: Update EXPOSE**
```dockerfile
# Before
EXPOSE 80

# After
EXPOSE 8080
```

**Step 3: Add non-root user**
```dockerfile
RUN useradd -r -u 999 appuser
USER appuser
```

**Step 4: Use port mapping at runtime**
```bash
docker run -p 80:8080 myapp
```

ALTERNATIVE: Use capabilities if port must stay privileged
```dockerfile
RUN setcap 'cap_net_bind_service=+ep' /usr/bin/myapp
USER appuser
EXPOSE 80
```

REFERENCES:
- Linux Capabilities: CAP_NET_BIND_SERVICE
- CIS Docker Benchmark: Section 4.5
- OWASP Docker Security Cheat Sheet
- RFC 6335: Internet Assigned Numbers Authority (IANA)
"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction


@dockerfile_rule(
    id="DOCKER-AUD-003",
    name="Privileged Port Exposed",
    severity="MEDIUM",
    cwe="CWE-250",
    category="audit",
    tags="docker,dockerfile,port,expose,privileged,root,security,unix,networking,capabilities,best-practice",
    message="Exposing port below 1024 typically requires root privileges to bind. Consider using non-privileged ports (>1024) with port mapping or granting CAP_NET_BIND_SERVICE capability."
)
def privileged_port():
    """
    Detects exposure of privileged ports (1-1023).

    Binding to privileged ports requires root privileges or CAP_NET_BIND_SERVICE
    capability, which conflicts with running containers as non-root users.
    """
    return instruction(
        type="EXPOSE",
        port_less_than=1024
    )
