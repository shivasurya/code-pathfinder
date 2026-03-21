FROM ubuntu:22.04

# Good: Only install required packages
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
      nginx \
      curl \
      ca-certificates && \
    # Clean up apt cache to further reduce size
    rm -rf /var/lib/apt/lists/*

# Result: Smaller, more secure image
