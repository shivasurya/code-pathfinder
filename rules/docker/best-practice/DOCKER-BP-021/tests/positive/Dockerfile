FROM ubuntu:22.04

# Bad: Missing -y flag
# Build will hang waiting for user confirmation
RUN apt-get update
RUN apt-get install nginx curl

# This will fail in CI/CD pipelines
