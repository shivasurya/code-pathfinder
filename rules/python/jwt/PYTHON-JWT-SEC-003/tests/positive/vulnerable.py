import jwt

# SEC-002: none algorithm (decode)
payload = jwt.decode(token, "", algorithms=["none"])

# SEC-003: unverified decode
data = jwt.decode(token, "secret", options={"verify_signature": False})
