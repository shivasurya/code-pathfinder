FROM cgr.dev/chainguard/go:latest AS builder

WORKDIR /app

COPY sast-engine .

ARG POSTHOG_WEB_ANALYTICS

ARG PROJECT_COMMIT

ENV POSTHOG_API_KEY=$POSTHOG_WEB_ANALYTICS

ARG PROJECT_VERSION

RUN echo "Building version ${PROJECT_VERSION} with commit ${PROJECT_COMMIT}"

RUN go mod download

RUN go build -ldflags="-s -w -X github.com/shivasurya/code-pathfinder/sast-engine/cmd.Version=${PROJECT_VERSION} -X github.com/shivasurya/code-pathfinder/sast-engine/cmd.GitCommit=${PROJECT_COMMIT} -X github.com/shivasurya/code-pathfinder/sast-engine/analytics.PublicKey=${POSTHOG_API_KEY}" -v -o pathfinder .

FROM cgr.dev/chainguard/wolfi-base:latest

WORKDIR /app

# Install Python 3.14 and pip for DSL execution
RUN apk add --no-cache \
    python3 \
    py3-pip

# Install Python DSL library for rule execution
RUN pip install --no-cache-dir codepathfinder

# Copy pathfinder binary from builder
COPY --from=builder /app/pathfinder /usr/bin/pathfinder

COPY entrypoint.sh /usr/bin/entrypoint.sh

RUN chmod +x /usr/bin/pathfinder

RUN chmod +x /usr/bin/entrypoint.sh

LABEL maintainer="shiva@shivasurya.me"
LABEL io.modelcontextprotocol.server.name="dev.codepathfinder/pathfinder"

ENTRYPOINT ["/usr/bin/entrypoint.sh"]