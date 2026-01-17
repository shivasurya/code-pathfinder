from flask import request
import pickle

def unsafe_deserialize():
    data = request.get_data()
    obj = pickle.loads(data)
    return obj
