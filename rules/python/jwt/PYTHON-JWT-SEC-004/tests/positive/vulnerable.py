import jwt

# Vulnerable: password stored in JWT payload (base64-encoded, not encrypted)
payload = {"username": "admin", "password": "secret123", "role": "superuser"}
token = jwt.encode(payload, "key", algorithm="HS256")

# Vulnerable: credentials in payload
user_token = jwt.encode({"email": "user@example.com", "password": "hunter2"}, SECRET, algorithm="HS256")
