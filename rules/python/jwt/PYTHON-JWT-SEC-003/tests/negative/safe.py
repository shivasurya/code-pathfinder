import jwt
import os

# SECURE: Secret from environment variable
SECRET = os.environ["JWT_SECRET_KEY"]

# SECURE: Use strong algorithm with proper secret
token = jwt.encode(
    {"user_id": 123, "role": "admin", "exp": datetime.utcnow() + timedelta(hours=1)},
    SECRET,
    algorithm="HS256"
)

# SECURE: Verify signature and specify allowed algorithms
payload = jwt.decode(
    token,
    SECRET,
    algorithms=["HS256"],  # Explicitly whitelist algorithms
    options={"require": ["exp", "iat"]}
)

# SECURE: Use RS256 with asymmetric keys for distributed systems
with open("private_key.pem", "rb") as f:
    private_key = f.read()
token = jwt.encode(payload, private_key, algorithm="RS256")

with open("public_key.pem", "rb") as f:
    public_key = f.read()
payload = jwt.decode(token, public_key, algorithms=["RS256"])
