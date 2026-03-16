# --- file: app.py ---
from flask import Flask, request
from utils import run_diagnostic

app = Flask(__name__)


@app.route('/diag')
def diagnostics():
    host = request.args.get('host')
    output = run_diagnostic(host)
    return output

# --- file: utils.py ---
import os
import subprocess


def run_diagnostic(target):
    cmd = "ping -c 3 " + target
    os.system(cmd)
    result = subprocess.check_output(cmd, shell=True)
    return result.decode()
