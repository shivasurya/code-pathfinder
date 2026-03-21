from cryptography.hazmat.primitives.ciphers.aead import AESGCM
import os

# SECURE: AES-256-GCM provides both confidentiality and integrity
key = AESGCM.generate_key(bit_length=256)
aesgcm = AESGCM(key)
nonce = os.urandom(12)
ct = aesgcm.encrypt(nonce, b"sensitive data", associated_data=None)

# Or use ChaCha20-Poly1305
from cryptography.hazmat.primitives.ciphers.aead import ChaCha20Poly1305
key = ChaCha20Poly1305.generate_key()
chacha = ChaCha20Poly1305(key)
ct = chacha.encrypt(nonce, b"sensitive data", associated_data=None)
