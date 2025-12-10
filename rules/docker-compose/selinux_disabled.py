"""
COMPOSE-SEC-010: SELinux Separation Disabled

Security Impact: MEDIUM
CWE: CWE-732 (Incorrect Permission Assignment for Critical Resource)

DESCRIPTION:
This rule detects docker-compose services that explicitly disable SELinux separation
by setting `security_opt: - label:disable`. SELinux provides mandatory access control
(MAC) that acts as an additional security layer beyond traditional discretionary access
control, significantly limiting the impact of container compromises.

SECURITY IMPLICATIONS:
Disabling SELinux separation removes a critical defense-in-depth layer:

1. **Container Escape Prevention**: SELinux confines containers even if they run as root,
   preventing access to host resources.

2. **Lateral Movement Blocking**: Prevents compromised containers from accessing other
   containers' files and networks.

3. **Host Protection**: Restricts container processes from interacting with the host
   system, even with root privileges.

4. **Zero-Day Mitigation**: Provides protection against unknown vulnerabilities by
   enforcing mandatory access controls.

Real-world impact without SELinux:
```bash
# Container running as root with label:disable
# Attacker exploits vulnerability to get shell

# Without SELinux, can access host filesystem
ls /var/lib/docker/volumes  # Can see other containers' data
cat /etc/shadow             # Can read host files

# With SELinux, all of above would be denied:
# Permission denied (SELinux MAC blocks access)
```

VULNERABLE EXAMPLE:
```yaml
version: '3'
services:
  web:
    image: nginx
    security_opt:
      - label:disable  # CRITICAL: Disables SELinux protection
    # Container can now bypass SELinux MAC controls
```

SECURE EXAMPLE - Default (Recommended):
```yaml
version: '3'
services:
  web:
    image: nginx
    # No security_opt - uses default SELinux labels (SECURE)
    # Container is confined by SELinux policy
```

SECURE EXAMPLE - Custom Labels:
```yaml
version: '3'
services:
  web:
    image: nginx
    security_opt:
      - label:type:container_t          # Use standard container type
      - label:level:s0:c100,c200         # Custom MCS labels for isolation
    # SELinux enabled with custom labels for fine-grained control
```

HOW SELINUX PROTECTS CONTAINERS:
1. **Process Confinement**: Container processes run in `container_t` type, which is
   restricted from accessing most host resources.

2. **Multi-Category Security (MCS)**: Each container gets unique category labels
   (e.g., `s0:c1,c2`) preventing containers from accessing each other's files.

3. **Volume Access Control**: Volumes must be labeled correctly for container access,
   preventing unauthorized data access.

4. **Capability Restrictions**: Even with capabilities, SELinux can block dangerous
   operations based on type enforcement.

Example SELinux denial (protection working):
```bash
# Container tries to access host file
cat /etc/shadow
cat: /etc/shadow: Permission denied

# SELinux audit log shows:
# type=AVC msg=audit(1234567890.123:456): avc: denied { read }
# for pid=1234 comm="cat" name="shadow" dev="sda1" ino=67890
# scontext=system_u:system_r:container_t:s0:c1,c2
# tcontext=system_u:object_r:shadow_t:s0
# tclass=file
```

WHEN LABEL:DISABLE MIGHT BE USED (Usually Wrong):
❌ "SELinux causes permission issues" - Fix labels instead of disabling
❌ "Need to access host filesystem" - Use proper volume mounts with correct labels
❌ "Performance concerns" - SELinux overhead is negligible in modern systems
❌ "Don't understand SELinux" - Learn it, don't disable it
✅ Non-SELinux systems (Ubuntu, Debian) - label:disable has no effect

PROPER SELinux TROUBLESHOOTING:
Instead of disabling, fix the root cause:

**Issue**: Container can't access volume
```bash
# WRONG: Disable SELinux
security_opt:
  - label:disable

# RIGHT: Label the volume correctly
volumes:
  - ./data:/data:z  # :z relabels for exclusive container access
  # or
  - ./data:/data:Z  # :Z relabels for shared access
```

**Issue**: Need specific file access
```bash
# Use semanage to create policy
sudo semanage fcontext -a -t container_file_t "/host/path(/.*)?"
sudo restorecon -Rv /host/path
```

**Issue**: Custom operations needed
```bash
# Create custom SELinux policy module instead of disabling
# Much more secure than blanket disable
```

DETECTION AND MONITORING:
```bash
# Check if SELinux is enabled for container
docker inspect <container> | grep -i selinux

# View SELinux denials
sudo ausearch -m avc -ts recent

# Check container's SELinux context
ps -eZ | grep containerd
```

REMEDIATION:
1. Remove `label:disable` from security_opt
2. If permission issues occur, relabel volumes with :z or :Z
3. For custom needs, create proper SELinux policies
4. Use `audit2allow` to generate policies from denials
5. Test in permissive mode before enforcing

COMBINING WITH OTHER SECURITY:
```yaml
version: '3'
services:
  hardened-app:
    image: myapp
    user: "1000:1000"
    security_opt:
      - no-new-privileges:true
      # SELinux enabled by default (no label:disable)
    read_only: true
    cap_drop:
      - ALL
    cap_add:
      - NET_BIND_SERVICE
```

REFERENCES:
- CWE-732: Incorrect Permission Assignment for Critical Resource
- https://www.redhat.com/en/topics/linux/what-is-selinux
- https://docs.docker.com/storage/bind-mounts/#configure-the-selinux-label
- SELinux Project: https://selinuxproject.org/
- OWASP A05:2021 - Security Misconfiguration
"""

from rules.container_decorators import compose_rule
from rules.container_matchers import service_has


@compose_rule(
    id="COMPOSE-SEC-012",
    name="SELinux Separation Disabled",
    severity="MEDIUM",
    cwe="CWE-732",
    category="security",
    message="Service has 'label:disable' in security_opt, which disables SELinux mandatory "
            "access control. This removes a critical security layer and increases the impact "
            "of container compromises. Remove label:disable or use custom SELinux labels instead."
)
def selinux_disabled():
    """
    Detects services that explicitly disable SELinux separation.

    Matches services with security_opt containing:
    - label:disable
    - label=disable
    """
    return service_has(
        key="security_opt",
        contains="label:disable"
    )
