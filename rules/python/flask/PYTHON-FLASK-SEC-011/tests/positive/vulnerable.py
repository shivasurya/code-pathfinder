from flask import Flask, request
import requests as http_requests

app = Flask(__name__)

@app.route('/api')
def api_call():
    host = request.args.get('host')
    url = "https://" + host + "/api/data"
    resp = http_requests.get(url)
    return resp.text
