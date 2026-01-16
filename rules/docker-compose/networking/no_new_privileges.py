"""
COMPOSE-SEC-009: Missing no-new-privileges Security Option

Security Impact: MEDIUM
CWE: CWE-732 (Incorrect Permission Assignment for Critical Resource)

DESCRIPTION:
This rule detects docker-compose services that do not set the `no-new-privileges:true`
security option. This option prevents processes in the container from gaining additional
privileges through setuid or setgid binaries, which can be used for privilege escalation attacks.

SECURITY IMPLICATIONS:
Without `no-new-privileges`, attackers can:

1. **Setuid Binary Exploitation**: Exploit setuid/setgid binaries to escalate from a
   low-privilege user to root within the container.

2. **Capability Escalation**: Use setuid binaries to acquire additional Linux capabilities
   that can be used for container escape.

3. **Binary Injection**: Replace legitimate setuid binaries with malicious versions that
   grant root access to attackers.

Real-world attack scenario:
```bash
# Attacker gains shell as www-data user
# Container has sudo with setuid bit
ls -la /usr/bin/sudo
-rwsr-xr-x 1 root root 157192 Jan 20  2021 /usr/bin/sudo

# Without no-new-privileges, attacker can escalate
/usr/bin/sudo /bin/bash
# Now root in container
```

Common setuid binaries that can be exploited:
- /usr/bin/sudo
- /usr/bin/su
- /usr/bin/passwd
- /bin/mount
- /usr/bin/newgrp
- Custom setuid binaries

VULNERABLE EXAMPLE:
```yaml
version: '3'
services:
  web:
    image: nginx
    # Missing no-new-privileges - VULNERABLE
    # Attacker can exploit setuid binaries
    user: www-data
```

SECURE EXAMPLE:
```yaml
version: '3'
services:
  web:
    image: nginx
    user: www-data
    security_opt:
      - no-new-privileges:true  # SECURE
    # Process cannot gain additional privileges
```

HOW IT WORKS:
The `no-new-privileges` flag is a Linux kernel feature (PR_SET_NO_NEW_PRIVS) that:
1. Prevents execve() from granting privileges via setuid/setgid bits
2. Blocks gaining capabilities through file capabilities
3. Inherited by child processes
4. Cannot be unset once enabled

Example of blocked escalation:
```bash
# With no-new-privileges enabled
$ id
uid=1000(appuser) gid=1000(appuser)

$ /usr/bin/sudo id
sudo: effective uid is not 0, is /usr/bin/sudo on a file system with the 'nosuid' option?

# Setuid bit ignored - privilege escalation blocked
```

WHEN TO USE:
- ✅ All production services
- ✅ Services running as non-root user
- ✅ Services that don't need privilege escalation
- ❌ Services that legitimately need setuid (very rare)

EXEMPTIONS (rare cases):
Some specialized containers may need setuid functionality:
- Containers that need to switch users (like sshd)
- Init system containers (systemd, s6)
- Development containers with sudo for convenience

For these cases, document the security exception and implement additional monitoring.

REMEDIATION:
```yaml
version: '3'
services:
  app:
    image: myapp
    user: "1000:1000"
    security_opt:
      - no-new-privileges:true
    # Additional hardening options
    read_only: true
    cap_drop:
      - ALL
    cap_add:
      - NET_BIND_SERVICE  # Only if needed
```

LAYERED SECURITY:
Combine with other security measures:
1. Drop all capabilities (`cap_drop: [ALL]`)
2. Run as non-root user
3. Use read-only filesystem
4. Remove setuid binaries from image
5. Use AppArmor/SELinux profiles

REFERENCES:
- CWE-732: Incorrect Permission Assignment for Critical Resource
- https://www.kernel.org/doc/Documentation/prctl/no_new_privs.txt
- https://raesene.github.io/blog/2019/06/01/docker-capabilities-and-no-new-privs/
- https://cheatsheetseries.owasp.org/cheatsheets/Docker_Security_Cheat_Sheet.html#rule-4-add-no-new-privileges-flag
- OWASP A05:2021 - Security Misconfiguration
"""

from rules.container_decorators import compose_rule
from rules.container_matchers import service_missing


@compose_rule(
    id="COMPOSE-SEC-011",
    name="Missing no-new-privileges Security Option",
    severity="MEDIUM",
    cwe="CWE-732",
    category="security",
    tags="docker-compose,compose,no-new-privileges,security,setuid,privilege-escalation,hardening,capabilities",
    message="Service does not have 'no-new-privileges:true' in security_opt. This allows "
            "processes to gain additional privileges via setuid/setgid binaries, which can be "
            "exploited for privilege escalation attacks."
)
def no_new_privileges():
    """
    Detects services missing the no-new-privileges security option.

    This check looks for services that either:
    1. Have no security_opt defined
    2. Have security_opt but don't include no-new-privileges:true
    """
    return service_missing(
        key="security_opt",
        value_contains="no-new-privileges:true"
    )
