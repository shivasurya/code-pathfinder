import jwt

# Vulnerable: hardcoded string as JWT signing secret
token = jwt.encode({"user": "admin"}, "hardcoded_secret", algorithm="HS256")

# Vulnerable: another hardcoded secret
auth_token = jwt.encode({"role": "superuser"}, "my_secret_key_123", algorithm="HS256")
