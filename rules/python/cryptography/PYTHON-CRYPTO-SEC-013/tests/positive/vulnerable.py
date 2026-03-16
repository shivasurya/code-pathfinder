from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.backends import default_backend

# SEC-010: MD5 in cryptography lib
digest = hashes.Hash(hashes.MD5(), backend=default_backend())

# SEC-011: SHA1 in cryptography lib
digest_sha1 = hashes.Hash(hashes.SHA1(), backend=default_backend())

# SEC-012: MD5 in PyCryptodome
from Crypto.Hash import MD5
h_md5 = MD5.new(b"data")

# SEC-013: MD4 in PyCryptodome
from Crypto.Hash import MD4
h_md4 = MD4.new(b"data")

# SEC-014: MD2 in PyCryptodome
from Crypto.Hash import MD2
h_md2 = MD2.new(b"data")

# SEC-015: SHA1 in PyCryptodome
from Crypto.Hash import SHA
h_sha = SHA.new(b"data")
