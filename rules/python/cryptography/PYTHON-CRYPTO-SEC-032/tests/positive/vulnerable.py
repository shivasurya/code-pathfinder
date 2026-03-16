from cryptography.hazmat.primitives.ciphers import Cipher, algorithms, modes
from cryptography.hazmat.backends import default_backend
from Crypto.Cipher import AES

aes_cbc = AES.new(key, AES.MODE_CBC, iv)
