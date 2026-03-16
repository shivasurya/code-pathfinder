from multiprocessing.connection import Client

conn = Client(('localhost', 6000))
data = conn.recv()
