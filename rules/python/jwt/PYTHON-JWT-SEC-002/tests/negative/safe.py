import jwt
import os

SECRET = os.environ["JWT_SECRET_KEY"]

# SAFE: Using HS256 algorithm
token = jwt.encode(
    {"user_id": 123, "role": "admin"},
    SECRET,
    algorithm="HS256"
)

# SAFE: Using RS256 with asymmetric keys
with open("private_key.pem", "rb") as f:
    private_key = f.read()
token = jwt.encode({"sub": "user123"}, private_key, algorithm="RS256")

# SAFE: Decode with proper algorithm whitelist
payload = jwt.decode(token, SECRET, algorithms=["HS256"])
