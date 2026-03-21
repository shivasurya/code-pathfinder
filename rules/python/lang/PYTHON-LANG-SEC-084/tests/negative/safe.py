import psycopg2
# Parameterized query with psycopg2 (safe)
cursor.execute("SELECT * FROM users WHERE id = %s", (user_id,))
# Parameterized query with asyncpg (safe)
await conn.fetch("SELECT * FROM users WHERE id = $1", user_id)
# Use psycopg2.sql module for dynamic identifiers
from psycopg2 import sql
query = sql.SQL("SELECT * FROM {} WHERE id = %s").format(sql.Identifier(table_name))
cursor.execute(query, (user_id,))
