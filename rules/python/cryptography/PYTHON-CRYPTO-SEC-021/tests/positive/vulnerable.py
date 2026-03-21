from cryptography.hazmat.primitives.asymmetric import rsa, dsa, ec
from cryptography.hazmat.backends import default_backend

# SEC-021: DSA key generation (audit)
dsa_key = dsa.generate_private_key(
    key_size=1024,
    backend=default_backend()
)
