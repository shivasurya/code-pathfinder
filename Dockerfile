FROM cgr.dev/chainguard/go:latest AS builder

WORKDIR /app

COPY sourcecode-parser .

ARG POSTHOG_WEB_ANALYTICS

ARG PROJECT_COMMIT

ENV POSTHOG_API_KEY=$POSTHOG_WEB_ANALYTICS

ARG PROJECT_VERSION

RUN echo "Building version ${PROJECT_VERSION} with commit ${PROJECT_COMMIT}"

RUN go mod download

RUN go build -ldflags="-s -w -X github.com/shivasurya/code-pathfinder/sourcecode-parser/cmd.Version=${PROJECT_VERSION} -X github.com/shivasurya/code-pathfinder/sourcecode-parser/cmd.GitCommit=${PROJECT_COMMIT} -X github.com/shivasurya/code-pathfinder/sourcecode-parser/analytics.PublicKey=${POSTHOG_API_KEY}" -v -o pathfinder .

FROM cgr.dev/chainguard/wolfi-base:latest

WORKDIR /app

COPY --from=builder /app/pathfinder /usr/local/bin/pathfinder

RUN chmod +x /usr/local/bin/pathfinder

CMD ["pathfinder", "version"]

LABEL maintainer="shiva@shivasurya.me"