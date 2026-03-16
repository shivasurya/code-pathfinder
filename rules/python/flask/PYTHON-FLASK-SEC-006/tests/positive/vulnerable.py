# --- file: app.py ---
from flask import Flask, request
from services import fetch_remote_data

app = Flask(__name__)


@app.route('/proxy')
def proxy():
    url = request.args.get('url')
    data = fetch_remote_data(url)
    return data

# --- file: services.py ---
import requests as http_requests


def fetch_remote_data(endpoint):
    resp = http_requests.get(endpoint)
    return resp.text
