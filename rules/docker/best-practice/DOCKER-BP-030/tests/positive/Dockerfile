FROM ubuntu:22.04

# Bad: cd in chain is confusing and pointless
RUN apt-get update && cd /tmp && apt-get install -y nginx
RUN mkdir /app; cd /app; touch file.txt
