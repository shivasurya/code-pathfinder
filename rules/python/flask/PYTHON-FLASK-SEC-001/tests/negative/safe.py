from flask import Flask, request
import subprocess
import shlex

app = Flask(__name__)

@app.route('/ping')
def ping_host():
    host = request.args.get('host')
    # SAFE: Use subprocess with list arguments (no shell interpretation)
    result = subprocess.run(['ping', '-c', '3', host], capture_output=True, text=True)
    return result.stdout

@app.route('/lookup')
def dns_lookup():
    domain = request.args.get('domain')
    # SAFE: shlex.quote() escapes shell metacharacters
    safe_domain = shlex.quote(domain)
    result = subprocess.check_output(['nslookup', safe_domain])
    return result
