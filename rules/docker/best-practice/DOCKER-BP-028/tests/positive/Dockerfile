FROM alpine:3.19

# Bad: Using apk upgrade
# Makes builds non-reproducible
RUN apk update && apk upgrade
RUN apk add nginx
