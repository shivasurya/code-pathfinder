import socket
import paramiko
import multiprocessing.connection

# SEC-070: bind to all interfaces
s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
s.bind(("0.0.0.0", 8080))

# SEC-071: paramiko implicit trust
client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())

# SEC-072: paramiko exec_command
stdin, stdout, stderr = client.exec_command("ls -la")

# SEC-073: multiprocessing recv
conn = multiprocessing.connection.Connection(1)
data = conn.recv()
