# Good: Digest pinning ensures immutable reference
FROM nginx:1.24.0@sha256:a4f34e6fb432af40bc594a0f1e5178598f6ca0f1ea6b3c6e7c5e5e8c3f9d6e1a

# Can combine readable tag with immutable digest
FROM ubuntu:22.04@sha256:b6b83d3c331794420340093eb706a6f152d9c1fa51b262d9bf34594887c2c7ac

# For multi-stage builds
FROM node:18@sha256:abc123... AS builder
FROM nginx:alpine@sha256:def456... AS runtime
