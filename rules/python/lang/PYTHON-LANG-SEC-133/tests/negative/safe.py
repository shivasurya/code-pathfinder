from cryptography.hazmat.primitives.asymmetric import padding, rsa
from cryptography.hazmat.primitives import hashes

# Safe: OAEP padding for RSA encryption (recommended)
def encrypt_message_oaep(public_key, message):
    ciphertext = public_key.encrypt(
        message,
        padding.OAEP(
            mgf=padding.MGF1(algorithm=hashes.SHA256()),
            algorithm=hashes.SHA256(),
            label=None
        )
    )
    return ciphertext

# Safe: PSS padding for RSA signatures (recommended)
def sign_data_pss(private_key, data):
    signature = private_key.sign(
        data,
        padding.PSS(
            mgf=padding.MGF1(hashes.SHA256()),
            salt_length=padding.PSS.MAX_LENGTH
        ),
        hashes.SHA256()
    )
    return signature

# Safe: PSS signature verification
def verify_pss(public_key, signature, data):
    public_key.verify(
        signature,
        data,
        padding.PSS(
            mgf=padding.MGF1(hashes.SHA256()),
            salt_length=padding.PSS.MAX_LENGTH
        ),
        hashes.SHA256()
    )

# Safe: Using Ed25519 (no padding needed)
from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey

def sign_ed25519(data):
    private_key = Ed25519PrivateKey.generate()
    signature = private_key.sign(data)
    return signature, private_key.public_key()
