"""
PYTHON-FLASK-SEC-013: Flask Insecure Deserialization

Security Impact: CRITICAL
CWE: CWE-502 (Deserialization of Untrusted Data)
OWASP: A08:2021 - Software and Data Integrity Failures

DESCRIPTION:
This rule detects insecure deserialization vulnerabilities in Flask applications where
user-controlled input flows into dangerous deserialization functions across multiple Python
serialization libraries. The following deserialization sinks are tracked:

- **pickle.loads() / pickle.load()**: Python's native binary serialization format. By
  design, pickle can instantiate arbitrary Python objects and execute code during
  deserialization via the __reduce__ protocol.

- **yaml.load() / yaml.unsafe_load()**: PyYAML's default loader can construct arbitrary
  Python objects from YAML tags (!!python/object, !!python/object/apply), enabling code
  execution through crafted YAML documents.

- **marshal.loads() / marshal.load()**: Python's internal serialization for .pyc files.
  Not intended for untrusted data and can cause crashes or undefined behavior.

- **jsonpickle.decode()**: JSON-based serialization that preserves Python object types,
  including the ability to reconstruct arbitrary objects during decoding.

- **shelve.open()**: Persistent dictionary backed by pickle, inheriting all pickle
  deserialization risks.

SECURITY IMPLICATIONS:

**1. Remote Code Execution**:
Pickle, YAML, and jsonpickle can all execute arbitrary Python code during deserialization.
An attacker crafts a serialized payload containing a class with a __reduce__ method (pickle)
or a !!python/object/apply tag (YAML) that calls os.system() or equivalent.

**2. System Compromise**:
Successful deserialization attacks provide the same privileges as the application process,
enabling attackers to read/write files, access databases, steal credentials, install
backdoors, and pivot to other systems.

**3. Denial of Service**:
Malformed serialized data can cause excessive memory allocation (billion laughs attack),
infinite loops, or application crashes through unexpected object reconstruction.

**4. Data Tampering**:
Attackers can reconstruct objects with manipulated attributes, bypassing business logic
validation that normally occurs during object creation.

VULNERABLE EXAMPLE:
```python
from flask import Flask, request
import pickle
import yaml

app = Flask(__name__)

@app.route('/api/import', methods=['POST'])
def import_data():
    data = request.get_data()
    # VULNERABLE: Untrusted data deserialized with pickle
    obj = pickle.loads(data)
    return {'imported': str(obj)}

@app.route('/api/config', methods=['POST'])
def load_config():
    config_yaml = request.form.get('config')
    # VULNERABLE: yaml.load() with default Loader allows code execution
    config = yaml.load(config_yaml)
    return {'config': config}
```

SECURE EXAMPLE:
```python
from flask import Flask, request
import json
import yaml

app = Flask(__name__)

@app.route('/api/import', methods=['POST'])
def import_data():
    # SAFE: Use JSON for untrusted data (cannot execute code)
    data = json.loads(request.get_data())
    return {'imported': data}

@app.route('/api/config', methods=['POST'])
def load_config():
    config_yaml = request.form.get('config')
    # SAFE: yaml.safe_load() only allows basic YAML types
    config = yaml.safe_load(config_yaml)
    return {'config': config}
```

DETECTION AND PREVENTION:

**Pre-deployment checks**:
```bash
pathfinder scan --project . --ruleset cpf/python/PYTHON-FLASK-SEC-013
```

**Code Review Checklist**:
- [ ] No pickle.loads()/load() with user-supplied data
- [ ] yaml.safe_load() used instead of yaml.load() for untrusted input
- [ ] json.loads() used as default deserialization for external data
- [ ] No jsonpickle.decode() with user-controlled input
- [ ] No shelve.open() with user-controlled paths
- [ ] HMAC signature verification applied before any internal pickle usage

COMPLIANCE:
- CWE-502: Deserialization of Untrusted Data
- OWASP Top 10 A08:2021 - Software and Data Integrity Failures
- SANS Top 25 (CWE-502 ranked in Top 25)
- PCI DSS Requirement 6.5.1: Injection Flaws

REFERENCES:
- CWE-502: https://cwe.mitre.org/data/definitions/502.html
- OWASP Deserialization Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Deserialization_Cheat_Sheet.html
- Python Pickle Security Warning: https://docs.python.org/3/library/pickle.html
- PyYAML Security: https://pyyaml.org/wiki/PyYAMLDocumentation
- OWASP A08:2021: https://owasp.org/Top10/A08_2021-Software_and_Data_Integrity_Failures/

DETECTION SCOPE:
This rule performs inter-procedural taint analysis tracking data from Flask request sources
to pickle, yaml, marshal, jsonpickle, and shelve deserialization sinks. Recognized sanitizers
include json.loads(), yaml.safe_load(), and hmac.compare_digest().
"""

from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets


class PickleModule(QueryType):
    fqns = ["pickle", "_pickle"]


class YamlModule(QueryType):
    fqns = ["yaml"]


class MarshalModule(QueryType):
    fqns = ["marshal"]


@python_rule(
    id="PYTHON-FLASK-SEC-013",
    name="Flask Insecure Deserialization",
    severity="CRITICAL",
    category="flask",
    cwe="CWE-502",
    tags="python,flask,deserialization,pickle,yaml,rce,owasp-a08,cwe-502",
    message="User input flows to unsafe deserialization (pickle/yaml). Use json.loads() or yaml.safe_load().",
    owasp="A08:2021",
)
def detect_flask_insecure_deserialization():
    """Detects Flask request data flowing to pickle.loads() or yaml.load()."""
    return flows(
        from_sources=[
            calls("request.get_data"),
            calls("request.get_json"),
            calls("request.form.get"),
            calls("request.args.get"),
        ],
        to_sinks=[
            PickleModule.method("loads", "load").tracks(0),
            YamlModule.method("load", "unsafe_load").tracks(0),
            MarshalModule.method("loads", "load").tracks(0),
            calls("jsonpickle.decode"),
            calls("shelve.open"),
        ],
        sanitized_by=[
            calls("json.loads"),
            calls("yaml.safe_load"),
            calls("hmac.compare_digest"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
