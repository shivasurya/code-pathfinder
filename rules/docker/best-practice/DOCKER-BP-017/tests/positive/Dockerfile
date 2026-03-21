FROM node:18

COPY . .

# Bad: cd doesn't persist, must chain all commands
RUN cd /app && npm install
RUN cd /app && npm build  # Must repeat cd
