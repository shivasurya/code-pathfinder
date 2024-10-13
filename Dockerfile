FROM cgr.dev/chainguard/go:latest AS builder

WORKDIR /app

COPY sourcecode-parser .

ARG POSTHOG_WEB_ANALYTICS

ENV POSTHOG_API_KEY=$POSTHOG_WEB_ANALYTICS

RUN PROJECT_VERSION=$(cat VERSION)

RUN GIT_COMMIT=$(git describe --tags)

RUN go mod download

RUN go build -ldflags="-s -w -X github.com/shivasurya/code-pathfinder/sourcecode-parser/cmd.Version=${PROJECT_VERSION} -X github.com/shivasurya/code-pathfinder/sourcecode-parser/cmd.GitCommit=${GIT_COMMIT} -X github.com/shivasurya/code-pathfinder/sourcecode-parser/analytics.PublicKey=${POSTHOG_API_KEY}" -v -o pathfinder .

FROM cgr.dev/chainguard/wolfi-base:latest

WORKDIR /app

COPY --from=builder /app/pathfinder /usr/local/bin/pathfinder

RUN chmod +x /usr/local/bin/pathfinder

CMD ["pathfinder", "version"]

LABEL maintainer="shiva@shivasurya.me"