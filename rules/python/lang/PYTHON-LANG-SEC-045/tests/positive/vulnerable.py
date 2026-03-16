import pickle
import yaml
import marshal
import shelve

# SEC-045: shelve
db = shelve.open("mydb")
