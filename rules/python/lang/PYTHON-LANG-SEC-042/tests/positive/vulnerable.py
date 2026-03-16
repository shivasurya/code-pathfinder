import pickle
import yaml
import marshal
import shelve

# SEC-042: jsonpickle
import jsonpickle
decoded = jsonpickle.decode('{"py/object": "os.system"}')
