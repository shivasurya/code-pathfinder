from flask import Flask, request
import sqlite3

app = Flask(__name__)

@app.route('/users/search')
def search_users():
    username = request.args.get('username')
    conn = sqlite3.connect('app.db')
    cursor = conn.cursor()
    # SAFE: Parameterized query prevents injection
    cursor.execute("SELECT * FROM users WHERE name = ?", (username,))
    return {'results': cursor.fetchall()}

# Alternative: Use an ORM like SQLAlchemy
from sqlalchemy import select
result = db.session.execute(select(User).filter_by(name=username))
