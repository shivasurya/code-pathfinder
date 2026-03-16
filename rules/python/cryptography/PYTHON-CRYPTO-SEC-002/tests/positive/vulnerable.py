from cryptography.hazmat.primitives.ciphers import Cipher, algorithms, modes
from cryptography.hazmat.backends import default_backend

# Blowfish has a 64-bit block size vulnerable to birthday attacks
bf_key = b'\x00' * 16
cipher = Cipher(algorithms.Blowfish(bf_key), modes.CBC(b'\x00' * 8), backend=default_backend())
