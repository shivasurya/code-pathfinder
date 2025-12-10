"""
COMPOSE-SEC-001: Service Running in Privileged Mode

Security Impact: CRITICAL
CWE: CWE-250 (Execution with Unnecessary Privileges)

DESCRIPTION:
This rule detects docker-compose services configured with `privileged: true`.
Privileged mode disables almost all container security features, granting the
container nearly all capabilities of the host machine. This is equivalent to
running as root on the host and can lead to complete host compromise.

WHAT PRIVILEGED MODE DOES:

When `privileged: true` is set, Docker:
1. Grants ALL Linux capabilities to the container (instead of limited subset)
2. Allows access to ALL host devices (/dev/*)
3. Disables AppArmor/SELinux confinement
4. Allows mounting any filesystem
5. Permits loading kernel modules
6. Enables access to host's PID namespace
7. Removes cgroup limitations

Essentially, it removes the container isolation boundary.

SECURITY IMPLICATIONS:

**1. Container Escape is Trivial**:
```bash
# Inside privileged container
docker run -it --privileged ubuntu bash

# Mount host filesystem
mkdir /host
mount /dev/sda1 /host

# Now have full read/write access to host
cat /host/etc/shadow
echo "attacker::0:0::/root:/bin/bash" >> /host/etc/passwd
```

**2. Kernel Module Loading**:
```bash
# Load malicious kernel module
insmod /tmp/rootkit.ko

# Kernel-level persistence, invisible to host
```

**3. Access All Host Devices**:
```bash
# Read raw disk
dd if=/dev/sda of=/tmp/disk.img

# Access hardware devices
# GPU, USB, network interfaces, etc.
```

**4. Bypass All Security Controls**:
- AppArmor profiles: Disabled
- SELinux policies: Disabled
- Seccomp filters: Disabled
- Capability restrictions: Removed

VULNERABLE EXAMPLE:
```yaml
version: '3.8'
services:
  # CRITICAL SECURITY ISSUE
  docker_runner:
    image: gitlab/gitlab-runner:latest
    privileged: true  # DANGEROUS!
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
```

This configuration:
- Gives CI runner unlimited host access
- Can create privileged containers
- Can mount host filesystem
- Can load kernel modules
- Bypass all security controls

If the CI runner is compromised:
```bash
# Attacker gains access to CI runner container
# Immediately escapes to host:
docker run --rm -it -v /:/host ubuntu chroot /host /bin/bash
# Now root on host machine
```

Real-world attack scenarios:
1. **Supply chain attack**: Malicious CI job loads rootkit
2. **Credential theft**: Mount /etc/shadow, steal SSH keys
3. **Lateral movement**: Access Docker socket, compromise all containers
4. **Data exfiltration**: Read all host files and volumes
5. **Crypto mining**: Use host resources for mining
6. **Ransomware**: Encrypt host filesystem

SECURE ALTERNATIVES:

**Solution 1: Use Specific Capabilities Instead**
```yaml
version: '3.8'
services:
  app:
    image: myapp:latest
    # DON'T use privileged: true
    # Instead, grant only needed capabilities
    cap_add:
      - NET_ADMIN      # If you need network config
      - SYS_NICE       # If you need to change process priority
    # Drop all other capabilities
    cap_drop:
      - ALL
```

Common capability use cases:
- `NET_ADMIN`: iptables, routing, VPN software
- `NET_RAW`: ping, traceroute
- `SYS_ADMIN`: mount filesystems (still dangerous, avoid if possible)
- `SYS_TIME`: Change system time (NTP servers)
- `SYS_NICE`: Adjust process priority

**Solution 2: Device Access Without Privileged**
```yaml
services:
  gpu_app:
    image: tensorflow:latest
    # Don't use privileged for GPU access
    devices:
      - /dev/nvidia0:/dev/nvidia0
      - /dev/nvidiactl:/dev/nvidiactl
      - /dev/nvidia-uvm:/dev/nvidia-uvm
```

**Solution 3: Docker-in-Docker Alternatives**
```yaml
# INSECURE: Traditional DinD
services:
  ci:
    image: docker:dind
    privileged: true  # AVOID THIS

# SECURE Alternative 1: Kaniko (no Docker daemon needed)
services:
  builder:
    image: gcr.io/kaniko-project/executor:latest
    # No privileged mode needed!

# SECURE Alternative 2: BuildKit with cache mounts
services:
  builder:
    image: moby/buildkit:latest
    # Uses rootless mode, no privileged needed
```

**Solution 4: Rootless Docker**
```yaml
services:
  docker:
    image: docker:dind-rootless
    # Runs Docker daemon as non-root user
    # No privileged mode required
    environment:
      DOCKER_TLS_CERTDIR: ""
```

LEGITIMATE USE CASES (RARE):

There are very few valid reasons to use privileged mode:

1. **True Docker-in-Docker** (consider alternatives first):
   ```yaml
   services:
     dind:
       image: docker:dind
       privileged: true
       # Only for CI/CD where DinD is absolutely required
   ```

2. **System-level tools** (monitoring, networking):
   ```yaml
   services:
     network_manager:
       image: custom-network-tool
       privileged: true
       # For tools that need to configure host networking
   ```

3. **Hardware access** requiring full device tree:
   ```yaml
   services:
     hardware_interface:
       image: industrial-control
       privileged: true
       # For specialized hardware control (PLCs, industrial systems)
   ```

**EVEN IN THESE CASES**: Document the security exception, implement monitoring,
and use network segmentation to limit blast radius.

DETECTION AND PREVENTION:

**Pre-deployment checks**:
```bash
# Scan docker-compose.yml for privileged mode
grep -n "privileged.*true" docker-compose.yml
```

**CI/CD gate**:
```yaml
# .github/workflows/security.yml
- name: Check for privileged containers
  run: |
    if grep -q "privileged:.*true" docker-compose.yml; then
      echo "ERROR: Privileged containers detected"
      exit 1
    fi
```

**Runtime enforcement** (Kubernetes):
```yaml
apiVersion: policy/v1beta1
kind: PodSecurityPolicy
metadata:
  name: restricted
spec:
  privileged: false  # Deny privileged pods
  allowPrivilegeEscalation: false
```

**Docker Bench Security**:
```bash
docker run --rm -it --net=host --pid=host \
  -v /var/run/docker.sock:/var/run/docker.sock \
  docker/docker-bench-security
# Checks for privileged containers
```

MONITORING AND DETECTION:

**Detect privileged containers**:
```bash
# List all privileged containers
docker ps --filter "label=com.docker.compose.service" --format "{{.Names}}: {{.Labels}}" | \
  while read line; do
    container=$(echo $line | cut -d: -f1)
    docker inspect $container | jq '.[0].HostConfig.Privileged'
  done
```

**Alert on privileged containers**:
```bash
# Falco rule (runtime security)
- rule: Privileged Container Started
  condition: container.privileged=true
  output: "Privileged container started (user=%user.name container=%container.id)"
  priority: CRITICAL
```

MIGRATION GUIDE:

**Step 1: Identify why privileged was used**
```bash
# Check container logs and behavior
docker-compose logs service_name
```

**Step 2: Replace with specific capabilities**
```yaml
# Before
services:
  app:
    privileged: true

# After
services:
  app:
    cap_add:
      - NET_ADMIN  # Or whatever specific capability is needed
    cap_drop:
      - ALL
```

**Step 3: Test thoroughly**
```bash
docker-compose up -d
docker-compose exec app /app/test.sh
# Verify functionality works without privileged mode
```

**Step 4: Add security context**
```yaml
services:
  app:
    security_opt:
      - no-new-privileges:true
      - apparmor:docker-default
    cap_drop:
      - ALL
```

COMPLIANCE AND AUDITING:

**CIS Docker Benchmark 5.4**:
> "Privileged containers should not be run"

**NIST SP 800-190**:
> "Containers should not be run in privileged mode except where absolutely necessary"

**SOC 2 / ISO 27001**:
Requires justification and approval for privileged access

**Audit logging**:
```yaml
services:
  audit:
    image: falcosecurity/falco
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    # Monitors for privileged container creation
```

REMEDIATION:

**Immediate action**:
```yaml
# Remove privileged flag
services:
  app:
    image: myapp
    # privileged: true  # DELETE THIS LINE
```

**If functionality breaks, use minimal privileges**:
```yaml
services:
  app:
    image: myapp
    cap_add:
      - NET_ADMIN  # Add only what's needed
    devices:
      - /dev/specific-device  # Mount only specific devices
```

REFERENCES:
- CWE-250: Execution with Unnecessary Privileges
- CIS Docker Benchmark: Section 5.4
- Docker Security Best Practices
- Linux Capabilities man page
- NIST SP 800-190: Container Security Guide
"""

from rules.container_decorators import compose_rule
from rules.container_matchers import service_has


@compose_rule(
    id="COMPOSE-SEC-001",
    name="Service Running in Privileged Mode",
    severity="CRITICAL",
    cwe="CWE-250",
    category="security",
    message="Service is running in privileged mode. This grants container equivalent of root capabilities on the host machine. Can lead to container escapes and privilege escalation."
)
def privileged_service():
    """
    Detects services with privileged: true.

    Privileged mode disables almost all container isolation, giving
    the container nearly all capabilities of the host. This is extremely
    dangerous and should be avoided except in very rare circumstances.
    """
    return service_has(
        key="privileged",
        equals=True
    )
