FROM ubuntu:22.04

RUN apt-get update && \
    apt-get install -y --no-install-recommends nginx && \
    rm -rf /var/lib/apt/lists/*

# Bad: Requires root to bind
EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
