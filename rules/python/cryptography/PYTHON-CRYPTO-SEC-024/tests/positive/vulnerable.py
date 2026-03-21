from cryptography.hazmat.primitives.asymmetric import rsa, dsa, ec
from cryptography.hazmat.backends import default_backend

# SEC-024: DSA in PyCryptodome (audit)
from Crypto.PublicKey import DSA
dsa_key_pc = DSA.generate(1024)
