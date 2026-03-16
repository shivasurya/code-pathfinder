import os
import socket

os.execl("/bin/sh", "sh", "-c", "echo hello")
