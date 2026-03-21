import pickle
import yaml
import marshal
import shelve

# SEC-046: dill
import dill
obj = dill.loads(b"data")
