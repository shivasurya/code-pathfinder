import pickle
import yaml
import marshal
import shelve

# SEC-040: pickle
data = pickle.loads(b"malicious")
with open("data.pkl", "rb") as f:
    obj = pickle.load(f)
unpickler = pickle.Unpickler(f)

# SEC-041: yaml unsafe load
with open("config.yml") as f:
    config = yaml.load(f, Loader=yaml.FullLoader)
    unsafe = yaml.unsafe_load(f)

# SEC-042: jsonpickle
import jsonpickle
decoded = jsonpickle.decode('{"py/object": "os.system"}')

# SEC-043: ruamel.yaml unsafe
from ruamel.yaml import YAML
ym = YAML(typ="unsafe")

# SEC-044: marshal
code_obj = marshal.loads(b"data")

# SEC-045: shelve
db = shelve.open("mydb")

# SEC-046: dill
import dill
obj = dill.loads(b"data")
