"""
COMPOSE-SEC-006: Container Filesystem is Writable

Security Impact: LOW
CWE: CWE-732 (Incorrect Permission Assignment for Critical Resource)

DESCRIPTION:
This rule detects services without `read_only: true` filesystem setting.
While containers have writable filesystems by default for compatibility,
making the root filesystem read-only significantly improves security by
preventing attackers from modifying binaries, installing malware, or
persisting backdoors within the container.

SECURITY BENEFITS OF READ-ONLY FILESYSTEMS:

1. **Prevents Malware Installation**:
   Attackers cannot write persistent backdoors or rootkits to the filesystem.

2. **Blocks Binary Modification**:
   Cannot replace legitimate binaries with trojanized versions.

3. **Immutable Infrastructure**:
   Enforces that containers are disposable and stateless.

4. **Reduces Attack Persistence**:
   Malware must remain in memory only, lost on container restart.

VULNERABLE EXAMPLE:
```yaml
services:
  web:
    image: nginx
    # Writable filesystem (default) - can be modified by attackers
```

SECURE EXAMPLE:
```yaml
services:
  web:
    image: nginx
    read_only: true
    tmpfs:
      - /tmp
      - /var/run
      - /var/cache/nginx
```

Note: Applications needing to write must use volumes or tmpfs for specific directories.

IMPLEMENTING READ-ONLY FILESYSTEMS:

**Simple Case (No Writes Needed)**:
```yaml
services:
  static_site:
    image: nginx-static
    read_only: true
```

**With Temporary Directories**:
```yaml
services:
  app:
    image: python-app
    read_only: true
    tmpfs:
      - /tmp:size=100M
      - /var/log:size=50M
```

**With Persistent Data**:
```yaml
services:
  database:
    image: postgres
    read_only: true
    volumes:
      - pgdata:/var/lib/postgresql/data
    tmpfs:
      - /tmp
      - /run
```

COMMON WRITABLE DIRECTORIES TO EXPOSE:

- `/tmp` - Temporary files
- `/var/log` - Application logs
- `/var/run` - PID files, sockets
- `/var/cache` - Application cache
- `/var/tmp` - Persistent temp files

REMEDIATION:
```yaml
services:
  app:
    read_only: true  # Add this
    tmpfs:
      - /tmp  # Add writable tmpfs as needed
```

REFERENCES:
- CWE-732: Incorrect Permission Assignment
- Docker Security Best Practices
- Immutable Infrastructure Principles
"""

from rules.container_decorators import compose_rule
from rules.container_matchers import service_missing


@compose_rule(
    id="COMPOSE-SEC-006",
    name="Container Filesystem is Writable",
    severity="LOW",
    cwe="CWE-732",
    category="security",
    message="Service has writable root filesystem. Consider making it read-only for better security."
)
def writable_filesystem():
    """
    Detects services without read_only: true.

    Read-only filesystems prevent attackers from modifying binaries,
    installing malware, or persisting backdoors.
    """
    return service_missing(key="read_only")
