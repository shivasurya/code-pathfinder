import jwt

# Vulnerable: encode with algorithm="none" disables signature
token = jwt.encode({"user": "admin"}, "", algorithm="none")

# Vulnerable: using none algorithm with payload
unsafe_token = jwt.encode({"role": "superuser"}, "key", algorithm="none")
