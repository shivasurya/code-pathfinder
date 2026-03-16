from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.backends import default_backend

# SEC-010: MD5 in cryptography lib
digest = hashes.Hash(hashes.MD5(), backend=default_backend())
