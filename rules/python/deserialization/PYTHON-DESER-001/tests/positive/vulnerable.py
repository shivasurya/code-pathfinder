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
