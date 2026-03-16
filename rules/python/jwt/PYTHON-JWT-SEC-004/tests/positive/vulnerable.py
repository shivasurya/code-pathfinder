import jwt

payload = {"username": "admin", "password": "secret123", "role": "superuser"}
token = jwt.encode(payload, "key", algorithm="HS256")
