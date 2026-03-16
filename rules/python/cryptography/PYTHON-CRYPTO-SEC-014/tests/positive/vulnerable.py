from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.backends import default_backend

# SEC-014: MD2 in PyCryptodome
from Crypto.Hash import MD2
h_md2 = MD2.new(b"data")
