"""
COMPOSE-SEC-003: Seccomp Confinement Disabled

Security Impact: HIGH
CWE: CWE-284 (Improper Access Control)

DESCRIPTION:
This rule detects services with seccomp (secure computing mode) disabled via
`security_opt: seccomp:unconfined`. Seccomp is a Linux kernel feature that
restricts which system calls a process can make. Disabling it allows containers
to use ALL system calls, significantly increasing the attack surface.

WHAT SECCOMP DOES:

Seccomp limits system calls that containers can make to the Linux kernel.
Docker's default seccomp profile blocks ~44 dangerous system calls out of 300+
available, including:

- `reboot`, `swapon`, `swapoff` - System control
- `mount`, `umount` - Filesystem operations
- `kexec_load` - Kernel loading
- `init_module`, `delete_module` - Kernel module operations
- `iopl`, `ioperm` - Direct hardware I/O
- `ptrace` - Process debugging/injection
- `acct` - Process accounting
- `add_key`, `request_key` - Kernel keyring

Without seccomp, attackers can:
1. Load malicious kernel modules (rootkits)
2. Debug and inject code into other processes
3. Modify system time to break audit logs
4. Access hardware devices directly
5. Escalate privileges through kernel exploits

VULNERABLE EXAMPLE:
```yaml
version: '3.8'
services:
  app:
    image: myapp
    security_opt:
      - seccomp:unconfined  # DANGEROUS - Allows all syscalls
```

ATTACK SCENARIOS:

1. **Kernel Module Loading**:
   ```bash
   # With seccomp disabled, attacker can load rootkit
   insmod /tmp/rootkit.ko
   ```

2. **Process Injection via ptrace**:
   ```bash
   # Inject code into another container's process
   ptrace(PTRACE_ATTACH, target_pid)
   ```

3. **Time Manipulation**:
   ```bash
   # Modify system time to break audit trails
   clock_settime(CLOCK_REALTIME, fake_time)
   ```

SECURE ALTERNATIVES:

**Default (Recommended)**:
```yaml
services:
  app:
    image: myapp
    # No security_opt needed - uses Docker's default seccomp profile
```

**Custom Seccomp Profile**:
```yaml
services:
  app:
    image: myapp
    security_opt:
      - seccomp=/path/to/custom-profile.json
```

Example custom profile:
```json
{
  "defaultAction": "SCMP_ACT_ERRNO",
  "architectures": ["SCMP_ARCH_X86_64"],
  "syscalls": [
    {
      "names": ["read", "write", "open", "close", "stat"],
      "action": "SCMP_ACT_ALLOW"
    }
  ]
}
```

LEGITIMATE USE CASES (RARE):

- Debugging tools requiring ptrace
- System monitoring tools
- Specialized hardware access

Even then, create a custom profile instead of disabling completely.

REMEDIATION:
```yaml
# Remove seccomp:unconfined
services:
  app:
    # security_opt: [seccomp:unconfined]  # DELETE THIS
```

REFERENCES:
- CWE-284: Improper Access Control
- Docker Seccomp Security Profiles
- Linux Seccomp BPF Documentation
"""

from rules.container_decorators import compose_rule
from rules.container_matchers import service_has


@compose_rule(
    id="COMPOSE-SEC-003",
    name="Seccomp Confinement Disabled",
    severity="HIGH",
    cwe="CWE-284",
    category="security",
    message="Service disables seccomp profile. Container can use all system calls, increasing attack surface."
)
def seccomp_disabled():
    """
    Detects services with seccomp disabled.

    Seccomp limits which system calls a container can make. Disabling
    it removes an important security layer and allows dangerous operations
    like kernel module loading and process injection.
    """
    return service_has(
        key="security_opt",
        contains="seccomp:unconfined"
    )
