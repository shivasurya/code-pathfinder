FROM ubuntu:22.04

# Good: Use WORKDIR to change directories
RUN apt-get update && apt-get install -y nginx
WORKDIR /app
RUN touch file.txt
