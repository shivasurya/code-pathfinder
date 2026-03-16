import os
import socket

# SEC-010: os.system
os.system("ls -la")
os.popen("cat /etc/passwd")

# SEC-011: os.exec*
os.execl("/bin/sh", "sh", "-c", "echo hello")
os.execvp("ls", ["ls", "-la"])

# SEC-012: os.spawn*
os.spawnl(os.P_NOWAIT, "/bin/sh", "sh")
os.spawnvp(os.P_WAIT, "ls", ["ls"])

# SEC-014: reverse shell pattern
s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
