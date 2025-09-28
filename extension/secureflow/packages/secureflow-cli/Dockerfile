# Use official Node.js slim image for smaller size and better security
FROM node:18-slim

# Install system dependencies needed for SecureFlow CLI
RUN apt-get update && apt-get install -y \
    curl \
    unzip \
    jq \
    git \
    ca-certificates \
    --no-install-recommends && \
    rm -rf /var/lib/apt/lists/* && \
    apt-get clean

# Create a non-root user for security
RUN groupadd -r secureflow && \
    useradd -r -g secureflow -d /home/secureflow -s /bin/bash -c "SecureFlow User" secureflow && \
    mkdir -p /home/secureflow/.secureflow && \
    mkdir -p /app/results && \
    chown -R secureflow:secureflow /home/secureflow /app

# Set working directory
WORKDIR /app

# Copy SecureFlow CLI files (only the necessary ones)
COPY --chown=secureflow:secureflow package*.json ./
COPY --chown=secureflow:secureflow bin/ ./bin/
COPY --chown=secureflow:secureflow lib/ ./lib/
COPY --chown=secureflow:secureflow config/ ./config/
COPY --chown=secureflow:secureflow profiler/ ./profiler/
COPY --chown=secureflow:secureflow scanner/ ./scanner/
COPY --chown=secureflow:secureflow tools/ ./tools/

# Install dependencies and link the CLI globally
RUN npm ci --only=production && \
    npm link && \
    npm cache clean --force

# Make the analysis script executable
RUN chmod +x /app/tools/mgmt/analyze-plugins.sh

# Switch to non-root user
USER secureflow

# Set environment variables
ENV HOME=/home/secureflow
ENV NODE_ENV=production
ENV TARGETS_FILE=/app/tools/mgmt/targets.txt
ENV MAX_PLUGINS=50

# Set default working directory for the script
WORKDIR /app

# Health check to ensure SecureFlow CLI is working
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD secureflow --help || exit 1

# Default command
CMD ["/bin/bash"]

# Labels for metadata
LABEL maintainer="SecureFlow Team"
LABEL description="Secure Docker container for WordPress plugin analysis using SecureFlow CLI"
LABEL version="1.0"
