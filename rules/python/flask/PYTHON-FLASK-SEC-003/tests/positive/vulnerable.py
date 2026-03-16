# --- file: app.py ---
from flask import Flask, request
from db import query_user

app = Flask(__name__)


@app.route('/user')
def get_user():
    username = request.args.get('username')
    result = query_user(username)
    return str(result)

# --- file: db.py ---
import sqlite3


def get_connection():
    return sqlite3.connect('app.db')


def query_user(name):
    conn = get_connection()
    cursor = conn.cursor()
    cursor.execute("SELECT * FROM users WHERE name = '" + name + "'")
    return cursor.fetchall()
