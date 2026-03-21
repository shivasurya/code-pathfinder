FROM python:3.11-slim

# CRITICAL VULNERABILITY: Secret in build arg
ARG API_KEY
ARG DATABASE_PASSWORD
ARG GITHUB_TOKEN
ARG AWS_SECRET_ACCESS_KEY

# These secrets are now permanently in the image!
RUN pip install --index-url=https://user:${GITHUB_TOKEN}@github.com/ my-private-package
RUN echo "DB_PASS=${DATABASE_PASSWORD}" > /app/config.ini
