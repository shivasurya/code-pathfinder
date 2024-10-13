FROM cgr.dev/chainguard/go:latest AS builder

WORKDIR /app

COPY sourcecode-parser .

RUN go mod download

RUN go build -o pathfinder .

FROM cgr.dev/chainguard/wolfi-base:latest

WORKDIR /app

COPY --from=builder /app/pathfinder /usr/local/bin/pathfinder

RUN chmod +x /usr/local/bin/pathfinder

CMD ["pathfinder", "version"]

LABEL maintainer="shiva@shivasurya.me"