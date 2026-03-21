from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.backends import default_backend

# SEC-013: MD4 in PyCryptodome
from Crypto.Hash import MD4
h_md4 = MD4.new(b"data")
