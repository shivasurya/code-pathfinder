"""DB layer with raw SQL — sink for tainted input from app.py."""
import sqlite3


def get_connection():
    return sqlite3.connect('app.db')


def query_user(name):
    conn = get_connection()
    cursor = conn.cursor()
    cursor.execute("SELECT * FROM users WHERE name = '" + name + "'")
    return cursor.fetchall()
