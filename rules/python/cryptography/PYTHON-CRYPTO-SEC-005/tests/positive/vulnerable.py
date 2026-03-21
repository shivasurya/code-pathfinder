from Crypto.Cipher import DES

des = DES.new(b'8byteky', DES.MODE_CBC, b'\x00' * 8)
