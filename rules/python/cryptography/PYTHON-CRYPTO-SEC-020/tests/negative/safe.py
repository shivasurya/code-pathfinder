from cryptography.hazmat.primitives.asymmetric import rsa, dsa, ec
from cryptography.hazmat.backends import default_backend

# SECURE: RSA-3072 or higher
private_key = rsa.generate_private_key(
    public_exponent=65537,
    key_size=3072,
    backend=default_backend()
)

# SECURE: DSA-2048 or higher
private_key = dsa.generate_private_key(
    key_size=2048,
    backend=default_backend()
)

# SECURE: Strong elliptic curve (P-256 or higher)
private_key = ec.generate_private_key(
    ec.SECP256R1(),  # NIST P-256
    backend=default_backend()
)

# SECURE: PyCryptodome RSA-3072
from Crypto.PublicKey import RSA
key = RSA.generate(3072)
