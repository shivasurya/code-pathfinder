"""
COMPOSE-SEC-009: Using Host PID Mode

Security Impact: HIGH
CWE: CWE-250 (Execution with Unnecessary Privileges)

DESCRIPTION:
This rule detects services using `pid: host`, which disables PID namespace
isolation. This allows the container to see and interact with ALL processes
running on the host system, including sending signals to them, viewing their
command lines, and potentially injecting code.

WHAT HOST PID MODE DOES:

With `pid: host`:
1. Container can see all host processes via `ps`, `/proc`, etc.
2. Can send signals (kill, SIGSTOP, etc.) to any host process
3. Can read /proc/<pid>/* for all processes
4. Accesses same PID namespace as host
5. Can enumerate running services and applications
6. Can discover security tools and monitoring agents

SECURITY IMPLICATIONS:

1. **Process Enumeration**:
   ```bash
   ps aux  # See ALL host processes including sensitive ones
   ```

2. **Signal Sending**:
   ```bash
   kill -9 <pid>  # Kill host processes (if permissions allow)
   killall -9 sshd  # DoS attack on SSH
   ```

3. **Information Disclosure**:
   ```bash
   cat /proc/*/cmdline  # Read command lines (may contain secrets)
   cat /proc/*/environ  # Read environment variables
   ```

4. **Process Injection** (with SYS_PTRACE):
   ```bash
   gdb -p <host_pid>  # Attach to host process
   ```

VULNERABLE EXAMPLE:
```yaml
services:
  monitor:
    image: monitoring-tool
    pid: host  # DANGEROUS - Can see and signal all host processes
```

Attack scenario:
```bash
# Inside container
ps aux | grep -i password  # Find processes with secrets in cmdline
cat /proc/$(pidof mysql)/environ  # Read MySQL env vars
kill -9 $(pidof fail2ban)  # Disable security tool
```

SECURE ALTERNATIVES:

**Default (Isolated PID Namespace)**:
```yaml
services:
  app:
    image: myapp
    # No pid: host - container has its own PID namespace
```

**Share PID Between Containers**:
```yaml
services:
  app:
    image: myapp
    pid: "service:other_container"  # Share with specific container only
```

**If Monitoring is Needed**:
Use dedicated monitoring agents on host, not in containers:
```bash
# On host (not in container)
docker stats
docker events
systemctl status
```

LEGITIMATE USE CASES (VERY RARE):

- System-level debugging tools (strace, gdb)
- Process monitoring (rare - better done on host)
- Init systems (systemd containers - uncommon)

Even in these cases, extreme caution is required.

REMEDIATION:
```yaml
# Remove host PID mode
services:
  app:
    # pid: host  # DELETE THIS
```

If process monitoring is genuinely needed:
- Run monitoring agent directly on host
- Use Docker API for container stats
- Use cAdvisor or Prometheus exporters

REFERENCES:
- CWE-250: Execution with Unnecessary Privileges
- Docker PID Namespace Documentation
- CIS Docker Benchmark: Section 5.15
"""

from rules.container_decorators import compose_rule
from rules.container_matchers import service_has


@compose_rule(
    id="COMPOSE-SEC-009",
    name="Using Host PID Mode",
    severity="HIGH",
    cwe="CWE-250",
    category="security",
    tags="docker-compose,compose,pid,host-pid,security,isolation,namespace,process,information-disclosure",
    message="Service uses host PID namespace. Container can see and potentially signal host processes."
)
def host_pid_mode():
    """
    Detects services using host PID namespace.

    Sharing the host PID namespace allows the container to view all
    host processes, send signals to them, and access sensitive process
    information via /proc.
    """
    return service_has(
        key="pid",
        equals="host"
    )
