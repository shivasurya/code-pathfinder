"""
DOCKER-SEC-005: Secret in Build Argument

Security Impact: CRITICAL
CWE: CWE-538 (Insertion of Sensitive Information into Externally-Accessible File or Directory)

DESCRIPTION:
This rule detects ARG instructions with names that suggest they contain sensitive
information such as passwords, API keys, tokens, or credentials. Build arguments are
permanently stored in the Docker image metadata and can be retrieved by anyone with
access to the image using 'docker history', making them unsuitable for secrets.

SECURITY IMPLICATIONS:
Build arguments (ARG) in Dockerfiles are fundamentally insecure for storing secrets:

1. Persistent in Image Layers: ARG values are baked into the image metadata and
   remain accessible even if the layer is deleted or the secret is "removed" later
2. Visible in Image History: Anyone with access to the image can run 'docker history'
   or 'docker inspect' to retrieve all build arguments
3. Supply Chain Exposure: Images pushed to registries expose these secrets to anyone
   who pulls the image
4. No Encryption: Build args are stored in plaintext within the image metadata
5. CI/CD Leakage: Build logs often echo ARG values, exposing them in CI/CD systems

Real-world attack scenario:
- Developer passes database password as build arg: --build-arg DB_PASSWORD=secret123
- Image is pushed to Docker Hub or private registry
- Attacker pulls image and runs: docker history image:tag --no-trunc
- Attacker extracts DB_PASSWORD from build history
- Attacker uses credentials to access production database

VULNERABLE EXAMPLE:
```dockerfile
FROM python:3.11-slim

# CRITICAL VULNERABILITY: Secret in build arg
ARG API_KEY
ARG DATABASE_PASSWORD
ARG GITHUB_TOKEN
ARG AWS_SECRET_ACCESS_KEY

# These secrets are now permanently in the image!
RUN pip install --index-url=https://user:${GITHUB_TOKEN}@github.com/ my-private-package
RUN echo "DB_PASS=${DATABASE_PASSWORD}" > /app/config.ini
```

Building this image:
```bash
docker build --build-arg API_KEY=sk_live_abc123 \
             --build-arg DATABASE_PASSWORD=supersecret \
             -t myapp:latest .
```

Extracting the secrets:
```bash
docker history myapp:latest --no-trunc
# Shows: ARG API_KEY=sk_live_abc123
# Shows: ARG DATABASE_PASSWORD=supersecret

docker inspect myapp:latest | grep -A 10 "Config"
# Reveals all build args in JSON metadata
```

SECURE ALTERNATIVES:

1. **Docker Secrets (Swarm/Kubernetes)**:
```dockerfile
FROM python:3.11-slim
# Secrets mounted at runtime, never in image
RUN --mount=type=secret,id=api_key \
    API_KEY=$(cat /run/secrets/api_key) pip install package
```

2. **BuildKit Secret Mounts**:
```dockerfile
# syntax=docker/dockerfile:1.4
FROM python:3.11-slim
RUN --mount=type=secret,id=github_token \
    pip install --index-url=https://user:$(cat /run/secrets/github_token)@github.com/ package
```

Build with:
```bash
docker buildx build --secret id=github_token,src=./token.txt -t myapp:latest .
```

3. **Environment Variables at Runtime**:
```dockerfile
FROM python:3.11-slim
# No secrets in build, pass at runtime
CMD python app.py
```

Run with:
```bash
docker run -e API_KEY="${API_KEY}" myapp:latest
```

4. **Multi-Stage Builds with Clean Final Stage**:
```dockerfile
# Stage 1: Use secret (this stage is discarded)
FROM python:3.11-slim AS builder
ARG GITHUB_TOKEN
RUN pip install --index-url=https://user:${GITHUB_TOKEN}@github.com/ package

# Stage 2: Clean final image without secrets
FROM python:3.11-slim
COPY --from=builder /usr/local/lib/python3.11/site-packages /usr/local/lib/python3.11/site-packages
# No ARG or secret in final image
```

DETECTED PATTERNS:
This rule detects ARG names containing (case-insensitive):
- password, passwd
- secret
- token, access_token, refresh_token
- key, apikey, api_key, private_key
- auth, authorization
- credential, cred
- client_secret, client_id (OAuth)
- access_key, secret_key (AWS)

BEST PRACTICES:
1. Never use ARG for secrets - use BuildKit secret mounts instead
2. Use multi-stage builds to isolate secrets to builder stages
3. Pass secrets at runtime via environment variables or volume mounts
4. Use dedicated secret management (HashiCorp Vault, AWS Secrets Manager)
5. Scan images for leaked secrets before pushing to registries
6. Implement registry scanning to detect accidental secret exposure

REMEDIATION:
Replace ARG-based secrets with BuildKit secret mounts:

```dockerfile
# Before (VULNERABLE)
ARG GITHUB_TOKEN
RUN pip install --index-url=https://${GITHUB_TOKEN}@github.com/ package

# After (SECURE)
# syntax=docker/dockerfile:1.4
RUN --mount=type=secret,id=github_token \
    pip install --index-url=https://$(cat /run/secrets/github_token)@github.com/ package
```

REFERENCES:
- CWE-538: Insertion of Sensitive Information into Externally-Accessible File
- Docker BuildKit Secret Mounts
- OWASP Docker Security Cheat Sheet
- NIST SP 800-190: Application Container Security Guide
"""

from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction


@dockerfile_rule(
    id="DOCKER-SEC-005",
    name="Secret in Build Argument",
    severity="CRITICAL",
    cwe="CWE-538",
    category="security",
    tags="docker,dockerfile,secrets,credentials,security,arg,build-arg,password,token,api-key,sensitive-data,information-disclosure",
    message="Build argument name suggests it contains a secret. ARG values are visible in image history via 'docker history'."
)
def secret_in_build_arg():
    """
    Detects ARG instructions with names suggesting secrets.

    Build arguments are stored in the image layer history and can be
    retrieved by anyone with access to the image. Never pass secrets
    as build arguments.
    """
    return instruction(
        type="ARG",
        arg_name_regex=r"(?i)^.*(password|passwd|secret|token|key|apikey|api_key|auth|credential|cred|private|access_token|client_secret).*$"
    )
