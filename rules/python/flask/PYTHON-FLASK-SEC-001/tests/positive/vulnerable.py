from flask import Flask, request
import os

app = Flask(__name__)

@app.route('/ping')
def ping_host():
    host = request.args.get('host')
    os.system("ping -c 1 " + host)
    return "done"
