from cryptography.hazmat.primitives.ciphers import Cipher, algorithms, modes
from cryptography.hazmat.backends import default_backend

# SEC-030: ECB mode
key = b'\x00' * 32
ecb_cipher = Cipher(algorithms.AES(key), modes.ECB(), backend=default_backend())

# SEC-031: CBC mode (unauthenticated - audit)
iv = b'\x00' * 16
cbc_cipher = Cipher(algorithms.AES(key), modes.CBC(iv), backend=default_backend())

# SEC-031: CTR mode (unauthenticated - audit)
ctr_cipher = Cipher(algorithms.AES(key), modes.CTR(iv), backend=default_backend())

# SEC-032: AES in PyCryptodome (audit)
from Crypto.Cipher import AES
aes_cbc = AES.new(key, AES.MODE_CBC, iv)
