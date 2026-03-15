"""
Network Security Rules for Python

Rules in this file:
- PYTHON-LANG-SEC-070: Socket Bind to All Interfaces (CWE-200)
- PYTHON-LANG-SEC-071: Paramiko Implicit Trust Host Key (CWE-322)
- PYTHON-LANG-SEC-072: Paramiko exec_command (CWE-78)
- PYTHON-LANG-SEC-073: multiprocessing Connection.recv() (CWE-502)

Security Impact: HIGH
CWE: CWE-322 (Key Exchange without Entity Authentication)
OWASP: A07:2021 - Identification and Authentication Failures

DESCRIPTION:
Network-facing Python applications must properly authenticate remote hosts,
restrict listening interfaces, sanitize remote commands, and safely handle
deserialized data from network connections. This module detects insecure socket
binding patterns, SSH host key verification bypass, unvalidated remote command
execution over SSH, and unsafe deserialization through multiprocessing connections.

SECURITY IMPLICATIONS:
Binding a socket to 0.0.0.0 exposes the service on all network interfaces,
including external-facing ones, when the service may only need localhost access.
Using paramiko's AutoAddPolicy or WarningPolicy accepts unknown SSH host keys
without verification, enabling man-in-the-middle attacks where an attacker
impersonates the target server and intercepts all SSH traffic including
credentials and commands. Passing unsanitized input to paramiko's exec_command()
allows OS command injection on the remote host. The multiprocessing
Connection.recv() method uses pickle internally to deserialize received objects,
which can execute arbitrary code if the connection source is untrusted.

    # Attack scenario: MITM via unverified SSH host key
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    # Attacker intercepts DNS and presents their own SSH server
    client.connect("production-server.internal")  # Connects to attacker's server
    stdin, stdout, stderr = client.exec_command("cat /etc/shadow")  # Sent to attacker

VULNERABLE EXAMPLE:
```python
import socket, paramiko
# Service exposed on all interfaces
sock = socket.socket()
sock.bind(("0.0.0.0", 8080))  # Accessible from external networks
# SSH without host key verification (MITM vulnerable)
client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
client.connect(host)
# Unsanitized remote command execution
client.exec_command(f"ls {user_input}")
# Pickle deserialization from network
conn = multiprocessing.connection.Client(("remote", 6000))
data = conn.recv()  # Deserializes with pickle
```

SECURE EXAMPLE:
```python
import socket, paramiko
# Bind to localhost only for internal services
sock = socket.socket()
sock.bind(("127.0.0.1", 8080))
# Verify SSH host keys explicitly
client = paramiko.SSHClient()
client.load_system_host_keys()
client.set_missing_host_key_policy(paramiko.RejectPolicy())
client.connect(host)
# Validate commands before remote execution
ALLOWED_COMMANDS = {"ls", "df", "uptime"}
if cmd not in ALLOWED_COMMANDS:
    raise ValueError("Unauthorized command")
client.exec_command(cmd)
# Use authenticated Pipe() for multiprocessing
parent_conn, child_conn = multiprocessing.Pipe()
```

DETECTION AND PREVENTION:
- Bind sockets to specific interfaces (127.0.0.1 for local, specific IP for services)
- Always use paramiko.RejectPolicy() and load known host keys
- Validate and sanitize all commands passed to exec_command()
- Use multiprocessing.Pipe() (authenticated) instead of raw connections
- Implement network segmentation to limit exposure of internal services

COMPLIANCE:
- CWE-200: Exposure of Sensitive Information to an Unauthorized Actor
- CWE-322: Key Exchange without Entity Authentication
- CWE-78: Improper Neutralization of Special Elements used in an OS Command
- CWE-502: Deserialization of Untrusted Data
- OWASP A05:2021 - Security Misconfiguration
- OWASP A07:2021 - Identification and Authentication Failures
- OWASP A08:2021 - Software and Data Integrity Failures

REFERENCES:
- https://cwe.mitre.org/data/definitions/322.html
- https://cwe.mitre.org/data/definitions/200.html
- https://cwe.mitre.org/data/definitions/502.html
- https://owasp.org/Top10/A07_2021-Identification_and_Authentication_Failures/
- https://docs.paramiko.org/en/stable/api/client.html
- https://docs.python.org/3/library/multiprocessing.html#multiprocessing.connection.Connection
- https://docs.python.org/3/library/socket.html#socket.socket.bind
"""

from rules.python_decorators import python_rule
from codepathfinder import calls, QueryType


class SocketModule(QueryType):
    fqns = ["socket"]


class ParamikoModule(QueryType):
    fqns = ["paramiko"]


@python_rule(
    id="PYTHON-LANG-SEC-070",
    name="Socket Bind to All Interfaces",
    severity="MEDIUM",
    category="lang",
    cwe="CWE-200",
    tags="python,socket,bind,network,cwe-200",
    message="Socket bound to 0.0.0.0 (all interfaces). Bind to specific interface in production.",
    owasp="A05:2021",
)
def detect_bind_all():
    """Detects socket.bind to 0.0.0.0 or empty string."""
    return calls("*.bind", match_position={"0[0]": "0.0.0.0"})


@python_rule(
    id="PYTHON-LANG-SEC-071",
    name="Paramiko Implicit Trust Host Key",
    severity="HIGH",
    category="lang",
    cwe="CWE-322",
    tags="python,paramiko,ssh,host-key,mitm,cwe-322",
    message="AutoAddPolicy/WarningPolicy trusts unknown host keys. Use RejectPolicy or verify keys.",
    owasp="A02:2021",
)
def detect_paramiko_trust():
    """Detects paramiko AutoAddPolicy and WarningPolicy usage."""
    return ParamikoModule.method("AutoAddPolicy", "WarningPolicy", "set_missing_host_key_policy")


@python_rule(
    id="PYTHON-LANG-SEC-072",
    name="Paramiko exec_command",
    severity="MEDIUM",
    category="lang",
    cwe="CWE-78",
    tags="python,paramiko,ssh,command-execution,cwe-78",
    message="paramiko exec_command() detected. Ensure command is not user-controlled.",
    owasp="A03:2021",
)
def detect_paramiko_exec():
    """Detects paramiko SSHClient.exec_command usage."""
    return calls("*.exec_command")


@python_rule(
    id="PYTHON-LANG-SEC-073",
    name="multiprocessing Connection.recv()",
    severity="MEDIUM",
    category="lang",
    cwe="CWE-502",
    tags="python,multiprocessing,recv,deserialization,cwe-502",
    message="Connection.recv() uses pickle internally. Not safe for untrusted connections.",
    owasp="A08:2021",
)
def detect_conn_recv():
    """Detects multiprocessing Connection.recv() which uses pickle."""
    return calls("*.recv", "multiprocessing.connection.Connection.recv")
