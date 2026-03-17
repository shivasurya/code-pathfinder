import jwt

# Vulnerable: decode with verify_signature=False bypasses integrity checks
data = jwt.decode(token, "secret", options={"verify_signature": False})

# Vulnerable: using options dict variable
opts = {"verify_signature": False, "verify_exp": False}
payload = jwt.decode(token, key, options=opts)
