FROM python:3.11-slim
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY . .
RUN addgroup -g 1000 appgroup && \
    adduser -D -u 1000 -G appgroup appuser
USER appuser:appgroup
EXPOSE 8080/tcp
HEALTHCHECK --interval=30s CMD wget --quiet --tries=1 --spider http://localhost:8080/health || exit 1
CMD ["python", "app.py"]
