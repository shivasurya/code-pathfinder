FROM ubuntu:22.04

# Create non-root user with specific UID/GID
RUN groupadd -r appuser -g 999 && \
    useradd -r -u 999 -g appuser appuser

# Install dependencies as root
RUN apt-get update && apt-get install -y nginx

# Switch to non-root user
USER appuser

# Copy application files (now owned by appuser)
COPY --chown=appuser:appuser app /app

CMD ["nginx", "-g", "daemon off;"]
