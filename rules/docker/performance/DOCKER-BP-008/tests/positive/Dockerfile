FROM python:3.11-slim

WORKDIR /app
COPY requirements.txt .

# Bad: Retains pip cache
RUN pip install -r requirements.txt

# Cache remains in /root/.cache/pip/
# Adds 50-200 MB depending on dependencies
