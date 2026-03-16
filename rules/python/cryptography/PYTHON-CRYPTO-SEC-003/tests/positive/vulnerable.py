from cryptography.hazmat.primitives.ciphers import Cipher, algorithms, modes
from cryptography.hazmat.backends import default_backend

idea_key = b'\x00' * 16
cipher = Cipher(algorithms.IDEA(idea_key), modes.CBC(b'\x00' * 8), backend=default_backend())
