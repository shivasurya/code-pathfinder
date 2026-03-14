import sqlite3
from flask import Flask, request

app = Flask(__name__)


@app.route("/search")
def search():
    query = request.args.get("q")          # SOURCE
    conn = sqlite3.connect("test.db")
    cursor = conn.cursor()
    cursor.execute("SELECT * FROM users WHERE name = '" + query + "'")  # SINK
    return cursor.fetchall()


@app.route("/safe")
def safe_search():
    query = request.args.get("q")
    conn = sqlite3.connect("test.db")
    cursor = conn.cursor()
    cursor.execute("SELECT * FROM users WHERE name = ?", (query,))  # SAFE
    return cursor.fetchall()


@app.route("/no-sql")
def no_sql():
    conn = sqlite3.connect("test.db")
    cursor = conn.cursor()
    cursor.execute("SELECT 1")  # NO SOURCE — not a vuln
    return "ok"


import hashlib


def hash_password_weak(password):
    return hashlib.md5(password.encode()).hexdigest()


def hash_password_strong(password):
    return hashlib.sha256(password.encode()).hexdigest()


import os


def set_permissions():
    os.chmod("/tmp/data", 0o777)


def set_safe_permissions():
    os.chmod("/tmp/data", 0o644)
