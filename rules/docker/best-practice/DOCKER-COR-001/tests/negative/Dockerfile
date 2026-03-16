FROM ubuntu:22.04

# Good: Single ENTRYPOINT per Dockerfile
ENTRYPOINT ["/app/start.sh"]

# Or use multi-stage builds if you need different entrypoints:
FROM ubuntu:22.04 AS stage1
ENTRYPOINT ["/app/build.sh"]

FROM ubuntu:22.04 AS stage2
ENTRYPOINT ["/app/start.sh"]
