import jwt

# SEC-001: hardcoded secret + SEC-004: encode audit
token = jwt.encode({"user": "admin"}, "my_secret_key", algorithm="HS256")

# SEC-002: none algorithm (encode)
unsafe_token = jwt.encode({"user": "admin"}, "", algorithm="none")

# SEC-002: none algorithm (decode)
payload = jwt.decode(token, "", algorithms=["none"])

# SEC-003: unverified decode
data = jwt.decode(token, "secret", options={"verify_signature": False})

# SEC-005: request data to jwt.encode (flow)
def create_token(request):
    user_data = request.args.get('user')
    return jwt.encode({"sub": user_data}, "key", algorithm="HS256")
