import aiopg

async def query_user(pool, user_input):
    async with pool.acquire() as conn:
        async with conn.cursor() as cursor:
            await cursor.execute("SELECT * FROM users WHERE name = '" + user_input + "'")
            return await cursor.fetchall()
