from Crypto.Cipher import ARC4

# PyCryptodome ARC4/RC4 — same broken cipher, different library
rc4 = ARC4.new(b'secret_key')
ciphertext = rc4.encrypt(b"secret data")
