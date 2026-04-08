from cryptography.hazmat.primitives.asymmetric import padding, rsa
from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.backends import default_backend

# SEC-133: RSA PKCS#1 v1.5 padding — vulnerable to Bleichenbacher's attack

# 1. RSA encryption with PKCS1v15 padding
def encrypt_message(public_key, message):
    ciphertext = public_key.encrypt(
        message,
        padding.PKCS1v15()
    )
    return ciphertext

# 2. RSA decryption with PKCS1v15 padding
def decrypt_message(private_key, ciphertext):
    plaintext = private_key.decrypt(
        ciphertext,
        padding.PKCS1v15()
    )
    return plaintext

# 3. RSA signing with PKCS1v15 padding
def sign_data(private_key, data):
    signature = private_key.sign(
        data,
        padding.PKCS1v15(),
        hashes.SHA256()
    )
    return signature

# 4. RSA signature verification with PKCS1v15
def verify_signature(public_key, signature, data):
    public_key.verify(
        signature,
        data,
        padding.PKCS1v15(),
        hashes.SHA256()
    )

# 5. PKCS1v15 stored in a variable
def get_padding():
    pad = padding.PKCS1v15()
    return pad

# 6. PKCS1v15 in JWT/token signing context
def create_jwt_signature(private_key, header_payload):
    return private_key.sign(
        header_payload.encode("utf-8"),
        padding.PKCS1v15(),
        hashes.SHA256()
    )
