import sqlite3

# SEC-080: psycopg2
import psycopg2
pg_conn = psycopg2.connect("dbname=test")
pg_cursor = pg_conn.cursor()
name = "user_input"
pg_cursor.execute("SELECT * FROM users WHERE name = '" + name + "'")
