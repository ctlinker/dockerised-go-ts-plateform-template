FROM tooling:base AS go-toolchain

ENV GO_VERSION=1.26.2

RUN curl -fsSL https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz \
    | tar -C /usr/local -xz

ENV PATH="/usr/local/go/bin:${PATH}"

# Pre-install/cache sqlc so it doesn't compile during service builds
# Fetch the pre-compiled binary directly
RUN curl -L -s https://github.com/sqlc-dev/sqlc/releases/download/v1.31.0/sqlc_1.31.0_linux_amd64.tar.gz | tar -xz -C /usr/local/bin

WORKDIR /workspace