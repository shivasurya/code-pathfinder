from cryptography.hazmat.primitives.ciphers.aead import AESGCM
import os

# SECURE: AES-GCM provides both confidentiality AND integrity
key = AESGCM.generate_key(bit_length=256)
aesgcm = AESGCM(key)
nonce = os.urandom(12)
ct = aesgcm.encrypt(nonce, plaintext, associated_data=b"header")
# Tampered ciphertext will raise InvalidTag on decrypt

# SECURE: PyCryptodome AES-GCM
from Crypto.Cipher import AES
cipher = AES.new(key, AES.MODE_GCM)
ct, tag = cipher.encrypt_and_digest(plaintext)

# SECURE: If CBC is required, add HMAC (Encrypt-then-MAC)
import hmac, hashlib
cipher = Cipher(algorithms.AES(key_enc), modes.CBC(iv))
ct = cipher.encryptor().update(plaintext) + cipher.encryptor().finalize()
mac = hmac.new(key_mac, ct, hashlib.sha256).digest()
# Verify MAC before decrypting
