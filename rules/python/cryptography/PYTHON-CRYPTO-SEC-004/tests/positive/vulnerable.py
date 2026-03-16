from Crypto.Cipher import ARC2

rc2 = ARC2.new(b'secret_key_16by', ARC2.MODE_CBC, b'\x00' * 8)
