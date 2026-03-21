FROM python:3.11-slim

WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

# No HEALTHCHECK - orchestrator cannot detect app failures
CMD ["python", "app.py"]
