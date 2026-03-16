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
