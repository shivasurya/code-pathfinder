FROM python:3.11-slim

WORKDIR /app

# Copy only requirements first for layer caching
COPY requirements.txt .

# Good: No cache retained
RUN pip install --no-cache-dir -r requirements.txt

# Copy application code
COPY . .

CMD ["python", "app.py"]
