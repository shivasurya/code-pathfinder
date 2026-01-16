"""
COMPOSE-SEC-008: Dangerous Capability Added

Security Impact: HIGH
CWE: CWE-250 (Execution with Unnecessary Privileges)

DESCRIPTION:
This rule detects services that add dangerous Linux capabilities via `cap_add`.
Linux capabilities divide root privileges into distinct units. Some capabilities
are extremely powerful and can be used for container escape or privilege escalation.
This rule flags the most dangerous ones.

DANGEROUS CAPABILITIES DETECTED:

1. **SYS_ADMIN** - Mount filesystems, namespace operations
   - Can mount host filesystems
   - Create new namespaces
   - Effectively equivalent to root

2. **NET_ADMIN** - Network configuration
   - Modify routing tables
   - Configure iptables/firewall
   - Create network tunnels

3. **SYS_PTRACE** - Trace/debug processes
   - Inject code into other containers
   - Read memory of any process
   - Bypass security controls

4. **SYS_MODULE** - Load/unload kernel modules
   - Install rootkits
   - Modify kernel behavior
   - Ultimate persistence

5. **DAC_READ_SEARCH** - Bypass file read permission checks
   - Read any file regardless of permissions
   - Access sensitive data

6. **ALL** - All capabilities
   - Nearly equivalent to privileged mode
   - Should never be used

VULNERABLE EXAMPLE:
```yaml
services:
  app:
    image: myapp
    cap_add:
      - SYS_ADMIN  # DANGEROUS - Can escape container
      - NET_ADMIN   # RISKY - Full network control
```

ATTACK SCENARIOS:

**With SYS_ADMIN**:
```bash
# Mount host filesystem
mkdir /mnt/host
mount /dev/sda1 /mnt/host
# Full host access achieved
```

**With SYS_PTRACE**:
```bash
# Inject code into other container
gdb -p <pid> --batch -ex "call system(\"/bin/bash -c 'evil command'\")"
```

**With SYS_MODULE**:
```bash
# Load rootkit
insmod /tmp/rootkit.ko
```

SECURE ALTERNATIVES:

**Principle of Least Privilege**:
```yaml
services:
  app:
    cap_drop:
      - ALL  # Drop all capabilities first
    cap_add:
      - NET_BIND_SERVICE  # Only add what's needed
      - CHOWN
```

**Safe Capabilities** (if needed):
- `NET_BIND_SERVICE` - Bind privileged ports
- `CHOWN` - Change file ownership
- `DAC_OVERRIDE` - Bypass file permission checks
- `FOWNER` - Bypass file owner checks
- `SET UID/SETGID` - Set user/group ID
- `KILL` - Send signals to processes

**For Network Operations** (instead of NET_ADMIN):
```yaml
# If you only need to bind privileged ports
cap_add:
  - NET_BIND_SERVICE  # Much safer than NET_ADMIN
```

REMEDIATION:
```yaml
services:
  app:
    cap_drop:
      - ALL
    cap_add:
      - NET_BIND_SERVICE  # Only add safe, necessary caps
```

REFERENCES:
- CWE-250: Execution with Unnecessary Privileges
- Linux Capabilities man page
- Docker Security Best Practices
"""

from rules.container_decorators import compose_rule
from rules.container_matchers import service_has


@compose_rule(
    id="COMPOSE-SEC-008",
    name="Dangerous Capability Added",
    severity="HIGH",
    cwe="CWE-250",
    category="security",
    tags="docker-compose,compose,capabilities,cap-add,security,privilege-escalation,container-escape,linux,kernel",
    message="Service adds dangerous capability. These capabilities can be used for container escape or privilege escalation."
)
def dangerous_capabilities():
    """
    Detects services with dangerous capabilities.

    Capabilities like SYS_ADMIN, SYS_MODULE, and SYS_PTRACE provide
    near-root powers and can be exploited for container escape.
    """
    return service_has(
        key="cap_add",
        contains_any=[
            "SYS_ADMIN",
            "NET_ADMIN",
            "SYS_PTRACE",
            "SYS_MODULE",
            "DAC_READ_SEARCH",
            "ALL"
        ]
    )
