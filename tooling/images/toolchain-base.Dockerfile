# This docker serve as base toolchain to be derived for the full|go|pnpm toolchains
FROM alpine:3.20

# Core utilities
RUN apk add --no-cache \
    bash \
    curl \
    git \
    ca-certificates \
    xz \
    unzip \
    build-base \
    libc6-compat

# Moon Setup
RUN curl -fsSL https://moonrepo.dev/install/moon.sh | bash
ENV PATH="/root/.moon/bin:${PATH}"

WORKDIR /workspace