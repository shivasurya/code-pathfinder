import uuid
import os
import re
import logging
import logging.config

# SEC-100: uuid1 (leaks MAC)
uid = uuid.uuid1()
