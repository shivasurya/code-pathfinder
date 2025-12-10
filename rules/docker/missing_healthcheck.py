"""
DOCKER-BP-022: Missing HEALTHCHECK Instruction

Security Impact: LOW
Category: Best Practice / Reliability

DESCRIPTION:
This rule detects Dockerfiles that do not include a HEALTHCHECK instruction.
Health checks allow Docker, Kubernetes, and other orchestrators to monitor
container health and automatically restart or replace failing containers,
improving application availability and resilience.

WITHOUT HEALTHCHECK:

When a container has no health check, orchestrators can only detect if the
container process is running, not if the application inside is actually
healthy and serving requests:

```bash
$ docker ps
CONTAINER ID   STATUS
abc123def456   Up 10 minutes
```

The container appears "Up" even if:
- The web server is deadlocked
- The database connections are exhausted
- The application is in a crash loop
- HTTP endpoints are returning 500 errors

WITH HEALTHCHECK:

```bash
$ docker ps
CONTAINER ID   STATUS
abc123def456   Up 10 minutes (healthy)
# Or...
abc123def456   Up 10 minutes (unhealthy)
```

Orchestrators can now:
1. **Detect actual failures** beyond process crashes
2. **Trigger automatic restarts** for unhealthy containers
3. **Route traffic away** from unhealthy instances
4. **Alert operators** when health degrades
5. **Gather metrics** on application availability

VULNERABLE EXAMPLE:
```dockerfile
FROM python:3.11-slim

WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

# No HEALTHCHECK - orchestrator cannot detect app failures
CMD ["python", "app.py"]
```

What can go wrong:
- App crashes but container stays running (zombie state)
- Database connection pool exhausted
- Memory leak causes slow degradation
- Deadlock in application logic
- External dependency failure (API, cache)

Container stays "Up" but serves 500 errors for hours!

SECURE EXAMPLE:
```dockerfile
FROM python:3.11-slim

WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

# Health check: Test HTTP endpoint every 30s
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD curl --fail http://localhost:8000/health || exit 1

CMD ["python", "app.py"]
```

Health check behavior:
- Runs every 30 seconds
- Waits 5 seconds after container start before first check
- Times out after 3 seconds
- Marks unhealthy after 3 consecutive failures

HEALTHCHECK SYNTAX:

```dockerfile
HEALTHCHECK [OPTIONS] CMD command

OPTIONS:
  --interval=DURATION (default: 30s)    # How often to run check
  --timeout=DURATION (default: 30s)     # Max time before considering check failed
  --start-period=DURATION (default: 0s) # Grace period before first check
  --retries=N (default: 3)              # Consecutive failures needed to mark unhealthy
```

HEALTH CHECK STRATEGIES BY APPLICATION TYPE:

**1. HTTP/REST API Services**:
```dockerfile
# Simple HTTP check
HEALTHCHECK --interval=30s --timeout=3s --retries=3 \
  CMD curl --fail http://localhost:8080/health || exit 1

# With specific status code
HEALTHCHECK CMD curl -f http://localhost:8080/health || exit 1

# HTTPS with cert validation skip (if using self-signed)
HEALTHCHECK CMD curl --fail --insecure https://localhost:8443/health || exit 1

# Using wget instead of curl
HEALTHCHECK CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1
```

**2. gRPC Services**:
```dockerfile
# Install grpc_health_probe
FROM golang:1.21 AS healthcheck
RUN GRPC_HEALTH_PROBE_VERSION=v0.4.24 && \
    wget -qO/bin/grpc_health_probe \
    https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/${GRPC_HEALTH_PROBE_VERSION}/grpc_health_probe-linux-amd64 && \
    chmod +x /bin/grpc_health_probe

FROM myapp:latest
COPY --from=healthcheck /bin/grpc_health_probe /bin/

HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
  CMD /bin/grpc_health_probe -addr=:50051 || exit 1
```

**3. Database Services**:
```dockerfile
# PostgreSQL
HEALTHCHECK --interval=10s --timeout=3s --retries=5 \
  CMD pg_isready -U postgres || exit 1

# MySQL
HEALTHCHECK --interval=10s --timeout=3s --retries=5 \
  CMD mysqladmin ping -h localhost || exit 1

# MongoDB
HEALTHCHECK --interval=10s --timeout=3s --retries=5 \
  CMD mongosh --eval "db.adminCommand('ping')" || exit 1

# Redis
HEALTHCHECK --interval=10s --timeout=3s --retries=5 \
  CMD redis-cli ping || exit 1
```

**4. Message Queues**:
```dockerfile
# RabbitMQ
HEALTHCHECK --interval=30s --timeout=10s --retries=5 \
  CMD rabbitmq-diagnostics ping || exit 1

# Kafka (if kafka-broker-api-versions available)
HEALTHCHECK --interval=30s --timeout=10s --retries=5 \
  CMD kafka-broker-api-versions --bootstrap-server localhost:9092 || exit 1
```

**5. Background Workers/Cron Jobs**:
```dockerfile
# Check if process is running and recent activity
HEALTHCHECK --interval=60s --timeout=5s --retries=3 \
  CMD pgrep -f worker.py && \
      test $(find /tmp/worker.heartbeat -mmin -2 | wc -l) -gt 0 || exit 1

# Application writes heartbeat file every minute
```

**6. Static File Servers (nginx)**:
```dockerfile
# Check nginx status
HEALTHCHECK --interval=30s --timeout=3s --retries=3 \
  CMD curl --fail http://localhost:80/ || exit 1

# Or check nginx process
HEALTHCHECK CMD pgrep nginx || exit 1
```

ADVANCED HEALTH CHECK PATTERNS:

**1. Comprehensive Health Endpoint**:
```python
# Flask example
from flask import Flask, jsonify
import psycopg2
import redis

app = Flask(__name__)

@app.route('/health')
def health_check():
    health = {
        "status": "healthy",
        "checks": {}
    }

    # Check database
    try:
        conn = psycopg2.connect(DATABASE_URL)
        conn.cursor().execute("SELECT 1")
        health["checks"]["database"] = "up"
    except Exception as e:
        health["status"] = "unhealthy"
        health["checks"]["database"] = f"down: {e}"

    # Check cache
    try:
        r = redis.Redis()
        r.ping()
        health["checks"]["redis"] = "up"
    except Exception as e:
        health["status"] = "unhealthy"
        health["checks"]["redis"] = f"down: {e}"

    status_code = 200 if health["status"] == "healthy" else 503
    return jsonify(health), status_code
```

Dockerfile:
```dockerfile
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
  CMD curl --fail http://localhost:8000/health || exit 1
```

**2. Shallow vs Deep Health Checks**:
```dockerfile
# Shallow: Just check if app responds (fast)
HEALTHCHECK --interval=10s --timeout=2s --retries=3 \
  CMD curl --fail http://localhost:8080/ping || exit 1

# Deep: Check dependencies (slower, more comprehensive)
# Use liveness (shallow) + readiness (deep) in Kubernetes instead
```

**3. Graceful Degradation**:
```python
@app.route('/health')
def health():
    # Return 200 even if some non-critical services are down
    # Only return 503 if core functionality is broken
    if database_connection_ok():
        return "OK", 200
    else:
        return "Database down", 503
```

KUBERNETES INTEGRATION:

Docker HEALTHCHECK translates to Kubernetes liveness probe:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: myapp
spec:
  containers:
  - name: myapp
    image: myapp:latest
    livenessProbe:
      httpGet:
        path: /health
        port: 8080
      initialDelaySeconds: 5    # --start-period
      periodSeconds: 30          # --interval
      timeoutSeconds: 3          # --timeout
      failureThreshold: 3        # --retries
    readinessProbe:
      httpGet:
        path: /ready
        port: 8080
      initialDelaySeconds: 5
      periodSeconds: 10
```

Best practice: Define both in Dockerfile AND Kubernetes manifests for defense-in-depth.

DOCKER COMPOSE INTEGRATION:

```yaml
version: '3.8'
services:
  web:
    image: myapp:latest
    healthcheck:
      test: ["CMD", "curl", "--fail", "http://localhost:8080/health"]
      interval: 30s
      timeout: 3s
      retries: 3
      start_period: 5s
    depends_on:
      db:
        condition: service_healthy  # Wait for DB to be healthy

  db:
    image: postgres:15
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 10s
      timeout: 3s
      retries: 5
```

COMMON PITFALLS:

**1. Too Aggressive Intervals**:
```dockerfile
# Bad: Checks every 5 seconds, high CPU overhead
HEALTHCHECK --interval=5s --timeout=1s CMD curl http://localhost:8080/health

# Good: Reasonable interval
HEALTHCHECK --interval=30s --timeout=3s CMD curl http://localhost:8080/health
```

**2. Missing curl/wget in Image**:
```dockerfile
# Bad: curl not installed
HEALTHCHECK CMD curl http://localhost:8080/health

# Good: Install curl or use alternative
RUN apt-get update && apt-get install -y --no-install-recommends curl
HEALTHCHECK CMD curl http://localhost:8080/health

# Alternative: Use nc (netcat)
HEALTHCHECK CMD nc -z localhost 8080 || exit 1

# Alternative: Custom script
COPY healthcheck.sh /
HEALTHCHECK CMD /healthcheck.sh
```

**3. Expensive Health Checks**:
```dockerfile
# Bad: Runs full test suite
HEALTHCHECK CMD pytest tests/ || exit 1

# Good: Lightweight endpoint
HEALTHCHECK CMD curl http://localhost:8080/ping || exit 1
```

**4. No Start Period**:
```dockerfile
# Bad: Checks immediately, marks unhealthy during startup
HEALTHCHECK --interval=10s CMD curl http://localhost:8080/health

# Good: Allows time for app to start
HEALTHCHECK --interval=10s --start-period=30s CMD curl http://localhost:8080/health
```

MONITORING AND OBSERVABILITY:

**View health status**:
```bash
docker ps
# Shows (healthy) or (unhealthy) in STATUS column

docker inspect --format='{{.State.Health.Status}}' container_id
# Output: healthy, unhealthy, or starting
```

**Health check logs**:
```bash
docker inspect --format='{{json .State.Health}}' container_id | jq
```

**Prometheus metrics** (if using cAdvisor):
```promql
container_health_status{name="myapp"}
```

REMEDIATION:

**Step 1: Add health endpoint to application**
```python
# Flask
@app.route('/health')
def health():
    return "OK", 200

# FastAPI
@app.get("/health")
def health():
    return {"status": "healthy"}
```

**Step 2: Add HEALTHCHECK to Dockerfile**
```dockerfile
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD curl --fail http://localhost:8000/health || exit 1
```

**Step 3: Test locally**
```bash
docker build -t myapp:test .
docker run -d --name test myapp:test
sleep 10
docker ps  # Should show (healthy)
docker inspect test | grep -A 10 Health
```

**Step 4: Monitor in production**
Set up alerts for containers marked unhealthy.

REFERENCES:
- Docker HEALTHCHECK documentation
- Kubernetes Liveness and Readiness Probes
- 12-Factor App: Admin Processes
- Production-Ready Health Checks (Microsoft)
"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import missing


@dockerfile_rule(
    id="DOCKER-BP-022",
    name="Missing HEALTHCHECK Instruction",
    severity="LOW",
    category="best-practice",
    message="No HEALTHCHECK instruction. Container health cannot be monitored by orchestrators, reducing reliability and observability."
)
def missing_healthcheck():
    """
    Detects missing HEALTHCHECK instruction.

    Health checks allow Docker, Kubernetes, and other orchestrators to
    monitor application health and automatically restart failing containers,
    significantly improving availability.
    """
    return missing(instruction="HEALTHCHECK")
