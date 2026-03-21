import hashlib

# SEC-032: hashlib.new with insecure algo
h = hashlib.new("md5", b"data")
