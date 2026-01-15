"""
DOCKER-BP-008: pip install Without --no-cache-dir

Security Impact: LOW
Category: Best Practice

DESCRIPTION:
This rule detects RUN instructions using `pip install` without the `--no-cache-dir`
flag. By default, pip caches downloaded packages and wheels in `~/.cache/pip/`,
which can add 50-200 MB to Docker images. The --no-cache-dir flag disables caching,
significantly reducing image size for Python applications.

PIP CACHING BEHAVIOR:

When pip installs packages, it caches:
1. **Downloaded packages** (.tar.gz, .whl files)
2. **Built wheels** (compiled from source distributions)
3. **HTTP responses** (package metadata)

Default cache locations:
- Linux: `/root/.cache/pip/`
- Wheel cache: `/root/.cache/pip/wheels/`
- HTTP cache: `/root/.cache/pip/http/`

IMAGE SIZE IMPACT:

```dockerfile
# Without --no-cache-dir
FROM python:3.11-slim
RUN pip install django requests numpy pandas
# Result: 450 MB image
# pip cache: ~120 MB in /root/.cache/pip/
```

```dockerfile
# With --no-cache-dir
FROM python:3.11-slim
RUN pip install --no-cache-dir django requests numpy pandas
# Result: 330 MB image
# Savings: 120 MB (27% reduction)
```

Real-world examples:
- Django + DRF + celery: 380 MB → 260 MB (32% reduction)
- FastAPI + uvicorn + sqlalchemy: 280 MB → 190 MB (32% reduction)
- Data science stack (numpy, pandas, scipy): 1.2 GB → 950 MB (21% reduction)
- Simple Flask app: 180 MB → 120 MB (33% reduction)

VULNERABLE EXAMPLE:
```dockerfile
FROM python:3.11-slim

WORKDIR /app
COPY requirements.txt .

# Bad: Retains pip cache
RUN pip install -r requirements.txt

# Cache remains in /root/.cache/pip/
# Adds 50-200 MB depending on dependencies
```

Cache breakdown for a typical Django app:
```
/root/.cache/pip/
├── http/           # ~30 MB (HTTP responses)
├── wheels/         # ~80 MB (built wheels)
└── selfcheck/      # <1 MB (pip version check)
```

SECURE EXAMPLE:
```dockerfile
FROM python:3.11-slim

WORKDIR /app

# Copy only requirements first for layer caching
COPY requirements.txt .

# Good: No cache retained
RUN pip install --no-cache-dir -r requirements.txt

# Copy application code
COPY . .

CMD ["python", "app.py"]
```

COMPREHENSIVE PYTHON DOCKERFILE BEST PRACTICES:

```dockerfile
FROM python:3.11-slim

# Set environment variables
ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PIP_NO_CACHE_DIR=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1

WORKDIR /app

# Install system dependencies (if needed)
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
      gcc \
      libpq-dev && \
    rm -rf /var/lib/apt/lists/*

# Create non-root user
RUN useradd -r -u 999 -m appuser

# Install Python dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy application
COPY --chown=appuser:appuser . .

# Switch to non-root user
USER appuser

CMD ["python", "app.py"]
```

ENVIRONMENT VARIABLES FOR PIP:

```dockerfile
# Disable cache via environment variable (global effect)
ENV PIP_NO_CACHE_DIR=1

# Disable pip version check (reduces HTTP calls)
ENV PIP_DISABLE_PIP_VERSION_CHECK=1

# Don't create .pyc files
ENV PYTHONDONTWRITEBYTECODE=1

# Force stdout/stderr to be unbuffered
ENV PYTHONUNBUFFERED=1
```

Benefits:
- `PIP_NO_CACHE_DIR=1`: Same as --no-cache-dir flag
- `PIP_DISABLE_PIP_VERSION_CHECK=1`: Avoids "pip is outdated" messages
- `PYTHONDONTWRITEBYTECODE=1`: No .pyc files → smaller image
- `PYTHONUNBUFFERED=1`: Real-time logs in Docker

MULTI-STAGE BUILD PATTERN:

```dockerfile
# Build stage: Install dependencies with build tools
FROM python:3.11-slim AS builder

WORKDIR /app

# Install build dependencies
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
      gcc \
      g++ \
      libpq-dev && \
    rm -rf /var/lib/apt/lists/*

# Install Python packages to custom location
COPY requirements.txt .
RUN pip install --no-cache-dir \
    --prefix=/install \
    -r requirements.txt

# Runtime stage: Copy only installed packages
FROM python:3.11-slim

# Copy installed packages from builder
COPY --from=builder /install /usr/local

# Copy application
WORKDIR /app
COPY . .

CMD ["python", "app.py"]
```

Benefits:
- Build tools (gcc, g++) not in final image
- Only compiled packages copied over
- Smallest possible runtime image

ADVANCED PIP OPTIMIZATION:

**1. Use pip-tools for Reproducible Builds**:
```dockerfile
# requirements.in (top-level dependencies)
django==4.2.7
djangorestframework==3.14.0

# Generate locked requirements.txt
# pip-compile requirements.in

# In Dockerfile
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
```

**2. Separate Dev and Prod Dependencies**:
```dockerfile
# Production
COPY requirements/prod.txt .
RUN pip install --no-cache-dir -r prod.txt

# Development
COPY requirements/dev.txt .
RUN pip install --no-cache-dir -r dev.txt
```

**3. Install from Wheels for Faster Builds**:
```dockerfile
# Pre-build wheels
RUN pip wheel --no-cache-dir \
    --wheel-dir=/wheels \
    -r requirements.txt

# Install from wheels (faster)
RUN pip install --no-cache-dir \
    --no-index \
    --find-links=/wheels \
    -r requirements.txt
```

**4. Use UV or Poetry for Faster Installs**:
```dockerfile
# Using UV (much faster than pip)
FROM python:3.11-slim
RUN pip install --no-cache-dir uv
COPY requirements.txt .
RUN uv pip install --system --no-cache -r requirements.txt
```

COMMON PITFALLS:

**1. Installing from Cache in Multi-Layer Builds**:
```dockerfile
# Bad: Each RUN creates a layer with cache
RUN pip install django
RUN pip install requests
RUN pip install celery
# 3 layers, each with partial cache

# Good: Single layer, no cache
RUN pip install --no-cache-dir \
    django \
    requests \
    celery
```

**2. Mixing System and User Installs**:
```dockerfile
# Bad: User install with cache
RUN pip install --user django

# Good: System install without cache
RUN pip install --no-cache-dir django
```

**3. Not Cleaning Up pip Itself**:
```dockerfile
# If you upgrade pip, clean up old version
RUN pip install --no-cache-dir --upgrade pip && \
    rm -rf /root/.cache/pip
```

VERIFICATION AND DEBUGGING:

**Check cache size**:
```bash
docker run --rm myimage du -sh /root/.cache/pip
```

**List installed packages**:
```bash
docker run --rm myimage pip list
```

**Verify no cache**:
```bash
docker run --rm myimage ls /root/.cache/
# Should not show 'pip' directory
```

**Compare image sizes**:
```bash
docker images myapp
# Before: 450 MB
# After: 330 MB
```

**Analyze layers**:
```bash
docker history myapp:latest
# Look for large layers from pip install
```

ALTERNATIVE APPROACHES:

**1. Distroless Python Images**:
```dockerfile
FROM gcr.io/distroless/python3-debian12
COPY --from=builder /usr/local/lib/python3.11/site-packages /usr/local/lib/python3.11/site-packages
```

**2. Alpine Python** (beware of musl libc issues):
```dockerfile
FROM python:3.11-alpine
RUN pip install --no-cache-dir django
```

**3. Custom Base Image with Pre-installed Packages**:
```dockerfile
# myorg/python-base:3.11
FROM python:3.11-slim
RUN pip install --no-cache-dir \
    django \
    djangorestframework \
    celery

# Application Dockerfile
FROM myorg/python-base:3.11
COPY . .
```

REMEDIATION STEPS:

**Step 1: Add --no-cache-dir flag**
```dockerfile
# Before
RUN pip install -r requirements.txt

# After
RUN pip install --no-cache-dir -r requirements.txt
```

**Step 2: Set environment variable (global)**
```dockerfile
ENV PIP_NO_CACHE_DIR=1
RUN pip install -r requirements.txt
```

**Step 3: Verify cleanup**
```bash
docker build -t myapp:test .
docker run --rm myapp:test ls /root/.cache/
# Should not show 'pip'
```

**Step 4: Measure improvement**
```bash
docker images myapp
# Compare before/after sizes
```

AUTOMATION:

**Hadolint Detection**:
```bash
hadolint Dockerfile
# DL3042: Avoid cache directory with pip install
```

**CI/CD Check**:
```yaml
- name: Verify pip best practices
  run: |
    grep -q "pip install --no-cache-dir" Dockerfile || \
    grep -q "PIP_NO_CACHE_DIR=1" Dockerfile || \
    exit 1
```

**Pre-commit Hook**:
```yaml
# .pre-commit-config.yaml
- repo: https://github.com/hadolint/hadolint
  hooks:
    - id: hadolint-docker
```

REFERENCES:
- pip documentation: Caching
- Docker Best Practices: Python Applications
- Python Official Docker Images Best Practices
- PEP 517: Build System Independence
"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction
from rules.container_combinators import all_of


@dockerfile_rule(
    id="DOCKER-BP-008",
    name="pip install Without --no-cache-dir",
    severity="LOW",
    category="best-practice",
    message="pip install without --no-cache-dir. Pip cache remains in image, adding 50-200 MB depending on dependencies."
)
def pip_without_no_cache():
    """
    Detects pip install without --no-cache-dir flag.

    pip caches downloaded packages in /root/.cache/pip/ which can
    add 50-200 MB to images. Use --no-cache-dir or ENV PIP_NO_CACHE_DIR=1.
    """
    return all_of(
        instruction(type="RUN", contains="pip install"),
        instruction(type="RUN", not_contains="--no-cache-dir")
    )
