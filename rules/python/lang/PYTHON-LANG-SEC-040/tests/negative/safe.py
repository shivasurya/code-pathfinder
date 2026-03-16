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
