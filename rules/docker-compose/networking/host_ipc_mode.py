"""
COMPOSE-SEC-010: Using Host IPC Mode

Security Impact: MEDIUM
CWE: CWE-250 (Execution with Unnecessary Privileges)

DESCRIPTION:
This rule detects services using `ipc: host`, which disables IPC (Inter-Process
Communication) namespace isolation. This allows the container to share shared
memory segments, semaphores, and message queues with the host system, potentially
enabling information disclosure and process interference.

WHAT HOST IPC MODE DOES:

With `ipc: host`:
1. Container shares System V IPC objects with host
2. Can access shared memory segments (shm)
3. Can access semaphores
4. Can access message queues
5. No IPC namespace isolation

SECURITY IMPLICATIONS:

1. **Shared Memory Access**:
   ```bash
   ipcs -m  # List all host shared memory segments
   # Can read data from other processes' shared memory
   ```

2. **Semaphore Manipulation**:
   ```bash
   ipcs -s  # List semaphores
   # Can potentially cause deadlocks or race conditions
   ```

3. **Message Queue Interception**:
   ```bash
   ipcs -q  # List message queues
   # Can read/modify IPC messages
   ```

4. **Information Disclosure**:
   Shared memory may contain sensitive data like:
   - Database connection pools
   - Session data
   - Cache contents
   - Cryptographic keys

VULNERABLE EXAMPLE:
```yaml
services:
  app:
    image: myapp
    ipc: host  # Shares IPC namespace with host
```

SECURE ALTERNATIVE:

**Default (Isolated IPC Namespace)**:
```yaml
services:
  app:
    image: myapp
    # No ipc: host - container has its own IPC namespace
```

**Share IPC Between Specific Containers**:
```yaml
services:
  app1:
    image: app1
  
  app2:
    image: app2
    ipc: "service:app1"  # Share IPC with app1 only, not host
```

LEGITIMATE USE CASES:

- X11 applications (GUI apps in containers)
- Applications specifically designed to share IPC with host
- Legacy applications with IPC dependencies

Even then, prefer container-to-container IPC sharing over host IPC.

REMEDIATION:
```yaml
# Remove host IPC mode
services:
  app:
    # ipc: host  # DELETE THIS
```

REFERENCES:
- CWE-250: Execution with Unnecessary Privileges
- Docker IPC Namespace Documentation
- System V IPC man pages
"""

from rules.container_decorators import compose_rule
from rules.container_matchers import service_has


@compose_rule(
    id="COMPOSE-SEC-010",
    name="Using Host IPC Mode",
    severity="MEDIUM",
    cwe="CWE-250",
    category="security",
    message="Service uses host IPC namespace. Container shares inter-process communication with host."
)
def host_ipc_mode():
    """
    Detects services using host IPC namespace.

    Sharing the host IPC namespace allows the container to access
    shared memory segments, semaphores, and message queues from
    host processes, potentially exposing sensitive data.
    """
    return service_has(
        key="ipc",
        equals="host"
    )
