from cryptography.hazmat.primitives.asymmetric import rsa, dsa, ec
from cryptography.hazmat.backends import default_backend

# SEC-023: RSA in PyCryptodome (audit)
from Crypto.PublicKey import RSA
rsa_key = RSA.generate(1024)
