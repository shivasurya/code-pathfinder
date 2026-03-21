from cryptography.hazmat.primitives.ciphers import Cipher, algorithms
from cryptography.hazmat.backends import default_backend

# ARC4/RC4 is a broken stream cipher with known biases
arc4_key = b'\x00' * 16
cipher = Cipher(algorithms.ARC4(arc4_key), mode=None, backend=default_backend())
encryptor = cipher.encryptor()
ciphertext = encryptor.update(b"secret data")
