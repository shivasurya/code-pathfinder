import msgpack
import ormsgpack

# SEC-139: msgpack.unpackb with ext_hook (code execution vector)
data = receive_from_network()
obj = msgpack.unpackb(data, ext_hook=custom_ext_hook)

# SEC-139: msgpack.unpack from file
with open("data.msgpack", "rb") as f:
    obj = msgpack.unpack(f)

# SEC-139: ormsgpack.unpackb (Rust-based msgpack, same risk)
result = ormsgpack.unpackb(untrusted_bytes)

# SEC-139: msgpack.Unpacker streaming deserialization
unpacker = msgpack.Unpacker(file_like_obj)
for msg in unpacker:
    process(msg)

# SEC-139: msgpack.unpackb without ext_hook (still flagged conservatively)
obj = msgpack.unpackb(raw_data)
