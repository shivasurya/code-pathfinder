# Vulnerable Dockerfile for testing security rules
# This should trigger multiple security findings

# DOCKER-BP-001: Using :latest tag
FROM ubuntu:latest

# DOCKER-BP-003: MAINTAINER deprecated
MAINTAINER test@example.com

# DOCKER-SEC-005: Secret in ARG
ARG DB_PASSWORD=changeme
ARG API_TOKEN
ARG GITHUB_AUTH_TOKEN

# DOCKER-BP-005: apt-get without --no-install-recommends
RUN apt-get update && apt-get install -y curl wget

# DOCKER-BP-007: apk without --no-cache
RUN apk add nodejs npm

# DOCKER-BP-008: pip without --no-cache-dir
RUN pip install requests flask

# DOCKER-AUD-003: Privileged port
EXPOSE 22
EXPOSE 80
EXPOSE 443

# DOCKER-SEC-006: Docker socket mount
VOLUME ["/var/run/docker.sock"]

# No HEALTHCHECK - DOCKER-BP-022

# DOCKER-SEC-001: No USER instruction - runs as root
CMD ["/bin/bash"]
