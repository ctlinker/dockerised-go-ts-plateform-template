FROM tooling:go AS builder

WORKDIR /repo

ENV MY_TARGET="infra/services/health"
COPY $MY_TARGET/go.mod $MY_TARGET/go.sum /repo/$MY_TARGET/

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd ./$MY_TARGET && go mod download

COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build \
    moon run service-health:build

CMD [ "moon", "run", "service-health:start" ]

# FROM repo/base AS base
# WORKDIR /app

# FROM base AS skeleton

# # Copy entire repository and scaffold
# COPY . .
# RUN moon docker scaffold service-health

# #### BUILD STAGE
# #### Builds the project.

# FROM base AS build

# # Copy workspace configs
# COPY --from=skeleton /app/.moon/docker/configs .

# # Install toolchains and dependencies
# # RUN moon docker setup

# # Copy project sources
# COPY --from=skeleton /app/.moon/docker/sources .

# # Build the project
# RUN moon run service-health:build

# # Prune extraneous dependencies
# RUN moon docker prune

# #### START STAGE
# #### Runs the project.

# FROM base AS start

# # Copy built sources
# COPY --from=build /root/.proto /root/.proto
# COPY --from=build /app /app

# CMD [ "moon", "run", "service-health:run" ]
