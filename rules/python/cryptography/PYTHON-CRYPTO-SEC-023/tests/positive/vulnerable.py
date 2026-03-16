from cryptography.hazmat.primitives.asymmetric import rsa, dsa, ec
from cryptography.hazmat.backends import default_backend

# SEC-020: RSA key generation (audit)
private_key = rsa.generate_private_key(
    public_exponent=65537,
    key_size=1024,
    backend=default_backend()
)

# SEC-021: DSA key generation (audit)
dsa_key = dsa.generate_private_key(
    key_size=1024,
    backend=default_backend()
)

# SEC-022: EC key generation (audit)
ec_key = ec.generate_private_key(
    ec.SECP192R1(),
    backend=default_backend()
)

# SEC-023: RSA in PyCryptodome (audit)
from Crypto.PublicKey import RSA
rsa_key = RSA.generate(1024)

# SEC-024: DSA in PyCryptodome (audit)
from Crypto.PublicKey import DSA
dsa_key_pc = DSA.generate(1024)
