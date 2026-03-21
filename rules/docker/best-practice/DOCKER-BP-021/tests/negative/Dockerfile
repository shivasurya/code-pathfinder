FROM ubuntu:22.04

# Good: Using -y flag for non-interactive installation
RUN apt-get update && \
    apt-get install -y nginx curl && \
    rm -rf /var/lib/apt/lists/*

# Alternative: Using --yes flag
RUN apt-get update && \
    apt-get install --yes \
        nginx \
        curl \
        ca-certificates && \
    rm -rf /var/lib/apt/lists/*
