import socket
import paramiko
import multiprocessing.connection

# SEC-072: paramiko exec_command
stdin, stdout, stderr = client.exec_command("ls -la")
