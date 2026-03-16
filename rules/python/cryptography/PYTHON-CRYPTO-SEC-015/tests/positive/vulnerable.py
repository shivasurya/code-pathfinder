from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.backends import default_backend

# SEC-015: SHA1 in PyCryptodome
from Crypto.Hash import SHA
h_sha = SHA.new(b"data")
