from Crypto.Cipher import XOR

xor = XOR.new(b'secret')
ciphertext = xor.encrypt(b'data')
