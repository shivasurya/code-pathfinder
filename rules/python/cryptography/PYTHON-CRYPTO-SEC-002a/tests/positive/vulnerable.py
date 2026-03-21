from Crypto.Cipher import Blowfish

bf = Blowfish.new(b'secret_key_16by', Blowfish.MODE_CBC, b'\x00' * 8)
