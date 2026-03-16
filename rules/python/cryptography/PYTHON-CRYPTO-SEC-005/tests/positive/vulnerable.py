from cryptography.hazmat.primitives.ciphers import Cipher, algorithms, modes
from cryptography.hazmat.backends import default_backend

# SEC-001: ARC4 cipher (cryptography lib)
arc4_key = b'\x00' * 16
cipher = Cipher(algorithms.ARC4(arc4_key), mode=None, backend=default_backend())

# SEC-002: Blowfish cipher (cryptography lib)
bf_key = b'\x00' * 16
cipher_bf = Cipher(algorithms.Blowfish(bf_key), modes.CBC(b'\x00' * 8), backend=default_backend())

# SEC-003: IDEA cipher (cryptography lib)
idea_key = b'\x00' * 16
cipher_idea = Cipher(algorithms.IDEA(idea_key), modes.CBC(b'\x00' * 8), backend=default_backend())

# SEC-001a: ARC4 in PyCryptodome
from Crypto.Cipher import ARC4
rc4 = ARC4.new(b'secret_key')

# SEC-002a: Blowfish in PyCryptodome
from Crypto.Cipher import Blowfish
bf = Blowfish.new(b'secret_key_16by', Blowfish.MODE_CBC, b'\x00' * 8)

# SEC-004: RC2 in PyCryptodome
from Crypto.Cipher import ARC2
rc2 = ARC2.new(b'secret_key_16by', ARC2.MODE_CBC, b'\x00' * 8)

# SEC-005: DES in PyCryptodome
from Crypto.Cipher import DES
des = DES.new(b'8byteky', DES.MODE_CBC, b'\x00' * 8)

# SEC-005a: Triple DES in PyCryptodome
from Crypto.Cipher import DES3
des3 = DES3.new(b'sixteen_byte_key_24b', DES3.MODE_CBC, b'\x00' * 8)

# SEC-006: XOR in PyCryptodome
from Crypto.Cipher import XOR
xor = XOR.new(b'secret')
