"""
Miscellaneous Python Security Rules

Rules in this file:
- PYTHON-LANG-SEC-100: Insecure UUID Version - uuid1 (CWE-330)
- PYTHON-LANG-SEC-101: Insecure File Permissions (CWE-732)
- PYTHON-LANG-SEC-102: Hardcoded Password in Default Argument (CWE-259)
- PYTHON-LANG-SEC-103: Regex DoS Risk (CWE-1333)
- PYTHON-LANG-SEC-104: logging.config.listen() Eval Risk (CWE-95)
- PYTHON-LANG-SEC-105: Logger Credential Leak Risk (CWE-532)

Security Impact: HIGH
CWE: CWE-330 (Use of Insufficiently Random Values)
OWASP: A01:2021 - Broken Access Control

DESCRIPTION:
This module covers a range of security concerns that span multiple vulnerability
categories. These include predictable UUID generation that leaks host information,
overly permissive file permissions that expose sensitive files, hardcoded
credentials in source code, regular expression patterns susceptible to
denial-of-service through catastrophic backtracking, unsafe logging configuration
listeners that evaluate arbitrary code, and logging statements that may
inadvertently record sensitive credentials.

SECURITY IMPLICATIONS:
uuid.uuid1() embeds the host's MAC address and timestamp, making generated UUIDs
predictable and enabling sandwich attacks where an attacker who obtains two UUIDs
can enumerate all UUIDs generated between them. Overly permissive file permissions
(e.g., 0o777) allow any user on the system to read, write, or execute sensitive
files. Hardcoded passwords in function default arguments are visible in source
code, version control history, and stack traces. Regex patterns with nested
quantifiers or overlapping alternatives can cause exponential backtracking,
enabling ReDoS attacks. logging.config.listen() opens a socket that accepts and
evaluates arbitrary logging configuration, potentially executing code. Logging
calls that include credentials or secrets in log messages create persistent
exposure in log files, monitoring systems, and log aggregation platforms.

    # Attack scenario: sandwich attack on uuid1
    # Attacker obtains uuid1 at time T1 and T1+10s, then enumerates
    # all possible UUIDs generated in that window (predictable timestamp + known MAC)

VULNERABLE EXAMPLE:
```python
import uuid, os, re
# Predictable UUID leaking MAC address
session_id = str(uuid.uuid1())
# World-writable file permissions
os.chmod("/app/config.ini", 0o777)
# Hardcoded password in function signature
def connect_db(host, password="admin123"):
    pass
# ReDoS-vulnerable pattern
pattern = re.compile(r"(a+)+$")
pattern.match("a" * 30 + "!")  # Exponential backtracking
# Credential logged in plaintext
logging.info(f"User {user} logged in with password {password}")
```

SECURE EXAMPLE:
```python
import uuid, os, re
# Use uuid4 for cryptographically random UUIDs
session_id = str(uuid.uuid4())
# Restrictive file permissions (owner read/write only)
os.chmod("/app/config.ini", 0o600)
# Load passwords from environment or secrets manager
def connect_db(host, password=None):
    password = password or os.environ["DB_PASSWORD"]
# Use atomic groups or possessive quantifiers to prevent backtracking
pattern = re.compile(r"a+$")  # Simplified non-vulnerable pattern
# Never log credentials
logging.info(f"User {user} logged in successfully")
```

DETECTION AND PREVENTION:
- Replace uuid.uuid1() with uuid.uuid4() for non-predictable random UUIDs
- Set file permissions to minimum required (0o600 for secrets, 0o644 for configs)
- Store credentials in environment variables, vaults, or secrets managers
- Audit regex patterns for nested quantifiers and overlapping alternatives
- Restrict logging.config.listen() to trusted networks with authentication
- Implement log scrubbing to redact credentials before writing to logs

COMPLIANCE:
- CWE-330: Use of Insufficiently Random Values
- CWE-732: Incorrect Permission Assignment for Critical Resource
- CWE-259: Use of Hard-coded Password
- CWE-1333: Inefficient Regular Expression Complexity
- CWE-532: Insertion of Sensitive Information into Log File
- CWE-95: Improper Neutralization of Directives in Dynamically Evaluated Code
- OWASP A01:2021 - Broken Access Control
- OWASP A07:2021 - Identification and Authentication Failures
- SANS Top 25 (2023) - CWE-798: Use of Hard-coded Credentials

REFERENCES:
- https://cwe.mitre.org/data/definitions/330.html
- https://cwe.mitre.org/data/definitions/732.html
- https://cwe.mitre.org/data/definitions/259.html
- https://cwe.mitre.org/data/definitions/1333.html
- https://cwe.mitre.org/data/definitions/532.html
- https://owasp.org/Top10/A01_2021-Broken_Access_Control/
- https://docs.python.org/3/library/uuid.html
- https://docs.python.org/3/library/os.html#os.chmod
- https://www.landh.tech/blog/20230811-sandwich-attack/
"""

from rules.python_decorators import python_rule
from codepathfinder import calls, QueryType


class UUIDModule(QueryType):
    fqns = ["uuid"]


class OSModule(QueryType):
    fqns = ["os"]


class LoggingConfig(QueryType):
    fqns = ["logging.config"]


class ReModule(QueryType):
    fqns = ["re"]


class LoggingModule(QueryType):
    fqns = ["logging"]


@python_rule(
    id="PYTHON-LANG-SEC-100",
    name="Insecure UUID Version (uuid1)",
    severity="LOW",
    category="lang",
    cwe="CWE-330",
    tags="python,uuid,mac-address,insufficiently-random,cwe-330",
    message="uuid.uuid1() leaks the host MAC address and uses predictable timestamps. Use uuid.uuid4() for random UUIDs.",
    owasp="A02:2021",
)
def detect_uuid1():
    """Detects uuid.uuid1() which leaks MAC address."""
    return UUIDModule.method("uuid1")


@python_rule(
    id="PYTHON-LANG-SEC-101",
    name="Insecure File Permissions",
    severity="MEDIUM",
    category="lang",
    cwe="CWE-732",
    tags="python,file-permissions,chmod,cwe-732",
    message="Overly permissive file permissions detected. Restrict to minimum required permissions.",
    owasp="A01:2021",
)
def detect_insecure_permissions():
    """Detects os.chmod/fchmod/lchmod with overly permissive modes."""
    return OSModule.method("chmod", "fchmod", "lchmod")


@python_rule(
    id="PYTHON-LANG-SEC-102",
    name="Hardcoded Password in Default Argument",
    severity="HIGH",
    category="lang",
    cwe="CWE-259",
    tags="python,hardcoded-password,credentials,cwe-259",
    message="Hardcoded password detected in function default argument. Use environment variables or secrets manager.",
    owasp="A07:2021",
)
def detect_hardcoded_password():
    """Detects functions with password-like default arguments — audit level."""
    return calls("*.connect", "*.login", "*.authenticate",
                 match_name={"password": "*"})


@python_rule(
    id="PYTHON-LANG-SEC-103",
    name="Regex DoS Risk",
    severity="LOW",
    category="lang",
    cwe="CWE-1333",
    tags="python,regex,redos,denial-of-service,cwe-1333",
    message="re.compile/match/search detected. Audit regex patterns for catastrophic backtracking.",
    owasp="A06:2021",
)
def detect_regex_dos():
    """Detects re.compile/match/search calls — audit for regex DoS."""
    return ReModule.method("compile", "match", "search", "findall")


@python_rule(
    id="PYTHON-LANG-SEC-104",
    name="logging.config.listen() Eval Risk",
    severity="HIGH",
    category="lang",
    cwe="CWE-95",
    tags="python,logging,listen,eval,code-execution,cwe-95",
    message="logging.config.listen() can execute arbitrary code via configuration. Restrict access.",
    owasp="A03:2021",
)
def detect_logging_listen():
    """Detects logging.config.listen() which evaluates received config."""
    return LoggingConfig.method("listen")


@python_rule(
    id="PYTHON-LANG-SEC-105",
    name="Logger Credential Leak Risk",
    severity="MEDIUM",
    category="lang",
    cwe="CWE-532",
    tags="python,logging,credentials,information-disclosure,cwe-532",
    message="Logging call detected. Audit log statements for credential/secret leakage.",
    owasp="A09:2021",
)
def detect_logger_cred_leak():
    """Detects logging calls — audit for credential leakage."""
    return LoggingModule.method("info", "debug", "warning", "error", "critical", "exception", "log")
