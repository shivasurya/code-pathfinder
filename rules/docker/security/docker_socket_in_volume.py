"""
DOCKER-SEC-006: Docker Socket Mounted as Volume

Security Impact: CRITICAL
CWE: CWE-250 (Execution with Unnecessary Privileges)

DESCRIPTION:
This rule detects VOLUME instructions that mount the Docker socket into a container.
Mounting the Docker socket (/var/run/docker.sock or /run/docker.sock) gives a container
full control over the host's Docker daemon, which is equivalent to unrestricted root
access on the host machine.

SECURITY IMPLICATIONS:
The Docker socket is owned by root and provides complete control over the Docker daemon.
When a container has access to this socket, it can:

1. **Container Escape**: Create privileged containers that mount the host filesystem:
   ```bash
   docker run -v /:/host --privileged alpine chroot /host /bin/sh
   ```
   This gives the attacker a root shell on the host.

2. **Privilege Escalation**: Start containers with any user ID, including UID 0 (root),
   and mount any host directory as a volume.

3. **Persistence**: Deploy malicious containers that persist across reboots by modifying
   host systemd services or cron jobs.

4. **Data Exfiltration**: Access all volumes, images, and containers on the host,
   including those containing sensitive data from other applications.

5. **Resource Hijacking**: Deploy cryptocurrency miners or consume all host resources
   to cause denial of service.

6. **Lateral Movement**: Access other containers' filesystems and networks, potentially
   compromising the entire infrastructure.

Real-world attack chain:
```bash
# Attacker gains shell in container with Docker socket mounted
# Step 1: List all containers
docker ps -a

# Step 2: Create privileged container mounting host root
docker run -it -v /:/host --privileged alpine /bin/sh

# Step 3: Chroot into host filesystem
chroot /host /bin/bash

# Step 4: Now has root access to host - install backdoor
echo "* * * * * root /tmp/backdoor.sh" >> /etc/crontab
```

VULNERABLE EXAMPLE:
```dockerfile
FROM docker:latest

# CRITICAL: Exposes Docker socket as volume
VOLUME ["/var/run/docker.sock"]

# This container can now control the host Docker daemon
CMD ["docker", "ps"]
```

Common vulnerable patterns in docker-compose:
```yaml
version: '3'
services:
  ci_runner:
    image: gitlab/gitlab-runner
    volumes:
      # CRITICAL: Gives CI runner full host access
      - /var/run/docker.sock:/var/run/docker.sock
```

WHY THIS IS DANGEROUS:
Unlike other security issues that require exploiting vulnerabilities, mounting the
Docker socket gives immediate, intentional root access. No exploit needed - it's
a feature being misused as an attack vector.

LEGITIMATE USE CASES:
There are a few scenarios where Docker socket mounting is necessary:

1. **Docker-in-Docker CI/CD**: Build and deploy containers from within containers
   - However, consider using Docker BuildKit remote builders or Kaniko instead
2. **Container Orchestration**: Tools like Portainer that manage Docker
   - Should be deployed with extreme caution and network isolation
3. **Monitoring/Logging**: Tools that inspect container metrics
   - Consider using read-only API access or cAdvisor instead

SECURE ALTERNATIVES:

**1. Docker-out-of-Docker (DooD) with Restricted Access:**
Instead of mounting the socket directly, use a proxy that restricts operations:
```yaml
services:
  docker-proxy:
    image: tecnativa/docker-socket-proxy
    environment:
      CONTAINERS: 1
      POST: 0  # Disable container creation
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock

  app:
    image: myapp
    environment:
      DOCKER_HOST: tcp://docker-proxy:2375  # Use proxy, not direct socket
```

**2. Kaniko for Building Images:**
```dockerfile
# Instead of Docker-in-Docker
FROM gcr.io/kaniko-project/executor:latest
# Builds containers without Docker daemon access
```

**3. Rootless Docker:**
Run Docker daemon as non-root user, limiting blast radius.

**4. Podman:**
Daemonless container engine that doesn't require socket access.

**5. Read-Only Docker API Access:**
If you only need to inspect containers:
```yaml
volumes:
  - /var/run/docker.sock:/var/run/docker.sock:ro  # Read-only
```

DETECTION AND PREVENTION:

**At Build Time:**
- Scan Dockerfiles for VOLUME instructions containing "docker.sock"
- Use this PathFinder rule in CI/CD pipelines
- Reject images that mount the Docker socket

**At Runtime:**
- Use admission controllers (OPA, Kyverno) to block containers mounting the socket
- Enable Docker Content Trust (DCT) to verify image signatures
- Implement least privilege through AppArmor/SELinux profiles

**Kubernetes Prevention:**
```yaml
apiVersion: policy/v1beta1
kind: PodSecurityPolicy
metadata:
  name: restrict-docker-socket
spec:
  volumes:
    - 'configMap'
    - 'emptyDir'
    - 'secret'
  # Omit 'hostPath' to prevent Docker socket mounting
```

REMEDIATION:
1. Remove VOLUME instruction mounting Docker socket
2. Evaluate if Docker access is truly necessary
3. If required, use restricted proxy or read-only access
4. Document the security exception and implement monitoring
5. Use network segmentation to limit container's network access

REFERENCES:
- CWE-250: Execution with Unnecessary Privileges
- Docker Socket Security Advisory
- NIST SP 800-190: Application Container Security Guide
- CIS Docker Benchmark: Section 5.31
"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction
from rules.container_combinators import any_of


@dockerfile_rule(
    id="DOCKER-SEC-006",
    name="Docker Socket Mounted as Volume",
    severity="CRITICAL",
    cwe="CWE-250",
    category="security",
    tags="docker,dockerfile,docker-socket,volume,security,privilege-escalation,container-escape,daemon,host-access,critical",
    message="Dockerfile mounts Docker socket. This gives the container full control over the host Docker daemon, equivalent to root access."
)
def docker_socket_in_volume():
    """
    Detects VOLUME instructions that include Docker socket paths.

    Mounting the Docker socket inside a container gives it full control
    over the Docker daemon, allowing it to create privileged containers,
    access host filesystem, and effectively become root on the host.
    """
    return any_of(
        instruction(type="VOLUME", contains="/var/run/docker.sock"),
        instruction(type="VOLUME", contains="/run/docker.sock"),
        instruction(type="VOLUME", contains="docker.sock")
    )
