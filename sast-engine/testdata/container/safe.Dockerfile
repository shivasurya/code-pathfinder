# Safe Dockerfile following security best practices
# This should NOT trigger security findings

# Pinned version tag
FROM ubuntu:22.04

# Use LABEL instead of MAINTAINER
LABEL maintainer="test@example.com"

# Safe ARG names (no secrets)
ARG APP_VERSION=1.0.0
ARG NODE_ENV=production

# apt-get with --no-install-recommends
RUN apt-get update && \
    apt-get install --no-install-recommends -y curl && \
    rm -rf /var/lib/apt/lists/*

# Non-privileged port
EXPOSE 8080

# HEALTHCHECK included
HEALTHCHECK --interval=30s --timeout=3s \
  CMD curl -f http://localhost:8080/health || exit 1

# Create non-root user
RUN useradd -m -u 1000 appuser

# Set working directory
WORKDIR /app

# Copy application files
COPY --chown=appuser:appuser . /app

# Switch to non-root user
USER appuser

# Application command
CMD ["./app"]
