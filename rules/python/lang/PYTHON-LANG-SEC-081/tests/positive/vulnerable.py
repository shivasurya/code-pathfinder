import sqlite3

# SEC-081: asyncpg
import asyncpg
async def query_asyncpg():
    conn = await asyncpg.connect("postgresql://localhost/test")
    await conn.execute("SELECT * FROM users WHERE id = " + user_id)
    await conn.fetch("SELECT * FROM t WHERE x = " + val)
