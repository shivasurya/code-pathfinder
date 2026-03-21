FROM alpine:3.19

# Bad: Leaves package cache in image
RUN apk update
RUN apk add \
    nginx \
    curl \
    ca-certificates

# Cache files remain in /var/cache/apk/*
# Adds 2-5 MB of unnecessary data
