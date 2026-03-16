from Crypto.Cipher import DES3

des3 = DES3.new(b'sixteen_byte_key_24b', DES3.MODE_CBC, b'\x00' * 8)
