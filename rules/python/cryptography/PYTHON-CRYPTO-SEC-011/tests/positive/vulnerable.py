from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.backends import default_backend

# SEC-011: SHA1 in cryptography lib
digest_sha1 = hashes.Hash(hashes.SHA1(), backend=default_backend())
