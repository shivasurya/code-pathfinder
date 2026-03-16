import pickle
import yaml
import marshal
import shelve

# SEC-044: marshal
code_obj = marshal.loads(b"data")
