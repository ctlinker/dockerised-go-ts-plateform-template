FROM tooling:go AS builder

WORKDIR /repo

ENV MY_TARGET="infra/gateway"

COPY $MY_TARGET/go.mod $MY_TARGET/go.sum /repo/$MY_TARGET/

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd ./$MY_TARGET && go mod download

COPY . .

RUN moon run gateway:build
CMD [ "moon", "run", "gateway:start" ]

