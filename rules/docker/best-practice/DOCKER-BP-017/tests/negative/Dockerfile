FROM node:18

# Good: WORKDIR persists across all subsequent instructions
WORKDIR /app
COPY . .
RUN npm install
RUN npm build  # Already in /app directory
