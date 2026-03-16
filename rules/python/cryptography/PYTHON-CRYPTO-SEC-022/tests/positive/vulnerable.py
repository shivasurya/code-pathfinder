from cryptography.hazmat.primitives.asymmetric import rsa, dsa, ec
from cryptography.hazmat.backends import default_backend

# SEC-022: EC key generation (audit)
ec_key = ec.generate_private_key(
    ec.SECP192R1(),
    backend=default_backend()
)
