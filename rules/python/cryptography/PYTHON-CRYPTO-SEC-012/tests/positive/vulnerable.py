from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.backends import default_backend

# SEC-012: MD5 in PyCryptodome
from Crypto.Hash import MD5
h_md5 = MD5.new(b"data")
