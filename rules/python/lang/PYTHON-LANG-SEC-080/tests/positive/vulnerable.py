import sqlite3

# SEC-080: psycopg2
import psycopg2
pg_conn = psycopg2.connect("dbname=test")
pg_cursor = pg_conn.cursor()
name = "user_input"
pg_cursor.execute("SELECT * FROM users WHERE name = '" + name + "'")

# SEC-081: asyncpg
import asyncpg
async def query_asyncpg():
    conn = await asyncpg.connect("postgresql://localhost/test")
    await conn.execute("SELECT * FROM users WHERE id = " + user_id)
    await conn.fetch("SELECT * FROM t WHERE x = " + val)

# SEC-084: formatted SQL (general)
conn2 = sqlite3.connect("test.db")
cursor = conn2.cursor()
cursor.execute("SELECT * FROM products WHERE id = " + product_id)
