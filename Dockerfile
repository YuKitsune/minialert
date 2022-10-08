# Build
FROM golang:1.19.1-alpine3.16 as build

ADD . /go/src/github.com/YuKitsune/minialert
WORKDIR /go/src/github.com/YuKitsune/minialert
RUN go build -o bin/minialert -ldflags "-X 'github.com/yukitsune/minialert.Version=$VERSION'" cmd/minialert/main.go

# Run
FROM alpine:3.16

# Versioning information
ARG GIT_COMMIT
ARG GIT_BRANCH=main
ARG GIT_DIRTY='false'
ARG VERSION
LABEL branch=$GIT_BRANCH \
    commit=$GIT_COMMIT \
    dirty=$GIT_DIRTY \
    version=$VERSION

COPY --from=build /go/src/github.com/YuKitsune/minialert/bin/minialert minialert

CMD  ["./minialert", "run"]