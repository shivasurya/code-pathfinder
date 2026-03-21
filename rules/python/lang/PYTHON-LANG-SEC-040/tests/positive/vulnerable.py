import pickle
import yaml
import marshal
import shelve

# SEC-040: pickle
data = pickle.loads(b"malicious")
with open("data.pkl", "rb") as f:
    obj = pickle.load(f)
unpickler = pickle.Unpickler(f)
