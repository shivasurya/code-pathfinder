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
