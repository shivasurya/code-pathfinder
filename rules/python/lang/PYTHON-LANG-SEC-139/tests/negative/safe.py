import json
import msgpack

# Safe: msgpack.packb (serialization, not deserialization)
packed = msgpack.packb({"key": "value"})

# Safe: json.loads for data interchange
data = json.loads(network_data)

# Safe: ormsgpack.packb (serialization only)
import ormsgpack
packed = ormsgpack.packb({"key": "value"})

# Safe: using pydantic for structured deserialization
from pydantic import BaseModel
class Config(BaseModel):
    name: str
    value: int

config = Config.model_validate_json(json_bytes)
