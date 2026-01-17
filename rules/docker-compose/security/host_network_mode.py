"""
COMPOSE-SEC-007: Using Host Network Mode

Security Impact: HIGH
CWE: CWE-250 (Execution with Unnecessary Privileges)

DESCRIPTION:
This rule detects services using `network_mode: host`, which disables network
isolation and makes the container share the host's network stack. This bypasses
Docker's network isolation, exposes all host network interfaces to the container,
and allows the container to bind to any host port.

WHAT HOST NETWORK MODE DOES:

With `network_mode: host`:
1. Container sees ALL host network interfaces
2. Can bind to ANY port on the host (including privileged ports <1024)
3. Bypasses Docker's port mapping and isolation
4. Has access to host's localhost services
5. Can sniff all network traffic on host
6. No network namespace separation

SECURITY IMPLICATIONS:

1. **Port Conflicts**: Container can bind to any host port, causing conflicts
2. **Service Exposure**: All container services exposed on host IP
3. **Network Sniffing**: Container can capture all host network traffic
4. **Localhost Access**: Can access services on host's 127.0.0.1
5. **No Firewall Protection**: Bypasses Docker's iptables rules

VULNERABLE EXAMPLE:
```yaml
services:
  app:
    image: myapp
    network_mode: host  # DANGEROUS - No network isolation
```

Attack scenario:
```bash
# Inside container with host network
tcpdump -i eth0  # Sniff all host network traffic
nc -l 22          # Hijack SSH port (if not already bound)
curl localhost:6379  # Access Redis on host if running
```

SECURE ALTERNATIVES:

**Use Bridge Network with Port Mapping**:
```yaml
services:
  app:
    image: myapp
    ports:
      - "8080:8080"  # Map specific ports only
    networks:
      - app_network

networks:
  app_network:
    driver: bridge
```

**Use Custom Network**:
```yaml
networks:
  backend:
    driver: bridge
    ipam:
      config:
        - subnet: 172.25.0.0/24

services:
  app:
    networks:
      backend:
        ipv4_address: 172.25.0.10
```

LEGITIMATE USE CASES (RARE):

- Network monitoring tools (Wireshark, tcpdump)
- VPN containers
- Network performance testing

Even then, consider using `NET_ADMIN` capability instead.

REMEDIATION:
```yaml
# Remove host network mode
services:
  app:
    # network_mode: host  # DELETE
    ports:
      - "8080:8080"  # Use explicit port mapping
```

REFERENCES:
- CWE-250: Execution with Unnecessary Privileges
- Docker Network Drivers Documentation
- CIS Docker Benchmark: Section 5.10
"""

from rules.container_decorators import compose_rule
from rules.container_matchers import service_has


@compose_rule(
    id="COMPOSE-SEC-007",
    name="Using Host Network Mode",
    severity="HIGH",
    cwe="CWE-250",
    category="security",
    tags="docker-compose,compose,network,host-network,security,isolation,networking,namespace,privilege-escalation",
    message="Service uses host network mode. Container shares host network stack, bypassing network isolation."
)
def host_network_mode():
    """
    Detects services using host network mode.

    Host network mode disables network namespace isolation, allowing
    the container to access all host network interfaces and localhost
    services, significantly increasing attack surface.
    """
    return service_has(
        key="network_mode",
        equals="host"
    )
