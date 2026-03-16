import hashlib

password = "user_password"
hashed = hashlib.md5(password.encode()).hexdigest()
