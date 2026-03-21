import hashlib

digest = hashlib.md5(b"data").hexdigest()
h = hashlib.md5()
h.update(b"more data")
