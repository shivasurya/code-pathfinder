import uuid
import os
import re
import logging
import logging.config

# SEC-102: hardcoded password
import mysql.connector
conn = mysql.connector.connect(host="db", password="secret123")
