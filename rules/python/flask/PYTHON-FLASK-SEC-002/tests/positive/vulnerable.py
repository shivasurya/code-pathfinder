from flask import Flask, request
import subprocess

app = Flask(__name__)

@app.route('/run')
def run_command():
    cmd = request.form.get('cmd')
    result = subprocess.check_output(cmd, shell=True)
    return result
