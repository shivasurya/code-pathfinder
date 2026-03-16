import uuid
import os
import re
import logging
import logging.config

# SEC-105: logger credential leak
logging.info("User logged in with password: %s", password)
logging.debug("API key: %s", api_key)
