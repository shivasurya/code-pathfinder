FROM ubuntu:22.04

# Good: Single CMD per Dockerfile
CMD ["nginx", "-g", "daemon off;"]

# Or use multi-stage builds if you need different commands:
FROM ubuntu:22.04 AS builder
CMD ["make", "build"]

FROM ubuntu:22.04 AS runtime
CMD ["nginx", "-g", "daemon off;"]
