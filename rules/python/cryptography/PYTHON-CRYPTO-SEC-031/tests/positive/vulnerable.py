from cryptography.hazmat.primitives.ciphers import Cipher, algorithms, modes
from cryptography.hazmat.backends import default_backend
from Crypto.Cipher import AES

# SEC-031: CBC mode (unauthenticated - audit)
cbc_cipher = Cipher(algorithms.AES(key), modes.CBC(iv), backend=default_backend())

aes_cbc = AES.new(key, AES.MODE_CBC, iv)
