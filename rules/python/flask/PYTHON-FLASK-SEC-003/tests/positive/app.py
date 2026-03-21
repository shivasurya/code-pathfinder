"""Cross-file SQL injection: source in routes, sink in db layer."""
from flask import Flask, request
from db import query_user

app = Flask(__name__)


@app.route('/user')
def get_user():
    username = request.args.get('username')
    result = query_user(username)
    return str(result)
