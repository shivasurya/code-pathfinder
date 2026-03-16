from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.backends import default_backend

# SECURE: SHA-256 for integrity checking
digest = hashes.Hash(hashes.SHA256(), backend=default_backend())
digest.update(document_bytes)
checksum = digest.finalize()

# SECURE: SHA-3 for digital signatures
from Crypto.Hash import SHA3_256
h = SHA3_256.new()
h.update(b"data to sign")
hash_value = h.hexdigest()

# SECURE: Argon2 for password hashing
from argon2 import PasswordHasher
ph = PasswordHasher()
password_hash = ph.hash(password)
