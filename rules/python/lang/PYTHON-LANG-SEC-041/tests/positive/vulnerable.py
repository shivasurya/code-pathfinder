import pickle
import yaml
import marshal
import shelve

# SEC-041: yaml unsafe load
with open("config.yml") as f:
    config = yaml.load(f, Loader=yaml.FullLoader)
    unsafe = yaml.unsafe_load(f)
