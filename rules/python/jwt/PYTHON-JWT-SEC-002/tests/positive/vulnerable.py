import jwt

token = jwt.encode({"user": "admin"}, "", algorithm="none")
decoded = jwt.decode(token, options={"verify_signature": False})
