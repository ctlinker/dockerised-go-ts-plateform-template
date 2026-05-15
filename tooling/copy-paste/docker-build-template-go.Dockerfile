FROM tooling:go

WORKDIR /repo

ENV MY_TARGET=""
COPY $MY_TARGET/go.mod $MY_TARGET/go.sum ./$MY_TARGET

RUN --mount=type=cache,target=/go/pkg/mod \
    cd ./$MY_TARGET && go mod download

COPY . .

# DO NOT FORGET TO CHANGE TARGET DIR
RUN go build -o app .