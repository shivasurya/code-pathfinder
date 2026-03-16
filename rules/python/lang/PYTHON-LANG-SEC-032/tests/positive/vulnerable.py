import hashlib

# SEC-030: MD5
md5_hash = hashlib.md5(b"data")

# SEC-031: SHA1
sha1_hash = hashlib.sha1(b"data")

# SEC-032: hashlib.new with insecure algo
h = hashlib.new("md5", b"data")

# SEC-033: SHA224
sha224_hash = hashlib.sha224(b"data")
sha3_224_hash = hashlib.sha3_224(b"data")

# SEC-034: MD5 for password
def hash_password(password):
    return hashlib.md5(password.encode()).hexdigest()
