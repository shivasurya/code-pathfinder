from cryptography.hazmat.primitives.ciphers import Cipher, algorithms, modes
from cryptography.hazmat.backends import default_backend
from Crypto.Cipher import AES

# SEC-030: ECB mode
ecb_cipher = Cipher(algorithms.AES(key), modes.ECB(), backend=default_backend())
