FROM ubuntu:latest

# Install packages without sudo (we're already root)
RUN apt-get update && apt-get install -y nginx \\
    && apt-get clean

# Create non-root user
RUN useradd -r -s /bin/false appuser

# Switch to non-root for runtime
USER appuser

# Run application as non-root
CMD ["nginx", "-g", "daemon off;"]
