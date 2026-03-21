FROM ubuntu:22.04

# Good: Removes package lists in same layer
RUN apt-get update && apt-get install -y nginx \\
    && rm -rf /var/lib/apt/lists/*
