import jwt

token = jwt.encode({"user": "admin"}, "hardcoded_secret", algorithm="HS256")
