FROM alpine:3.19

# Good: No cache retained
RUN apk add --no-cache \
    nginx \
    curl \
    ca-certificates

# Even better: Pin versions
RUN apk add --no-cache \
    nginx=1.24.0-r15 \
    curl=8.4.0-r0 \
    ca-certificates=20230506-r0
