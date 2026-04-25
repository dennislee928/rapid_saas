# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.22
ARG SERVICE_DIR=.
ARG CMD_DIR=./cmd/router-api

FROM golang:${GO_VERSION}-alpine AS build
ARG SERVICE_DIR
ARG CMD_DIR
WORKDIR /src

RUN apk add --no-cache ca-certificates git

COPY . .
WORKDIR /src/${SERVICE_DIR}

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/router ${CMD_DIR}

FROM alpine:3.20 AS runtime
RUN apk add --no-cache ca-certificates tzdata && adduser -D -H -u 10001 app
USER app
WORKDIR /app

COPY --from=build /out/router /app/router
COPY 05-infra-ci/root/router.config.example.yaml /app/router.config.yaml

ENV HTTP_ADDR=:8080
EXPOSE 8080

ENTRYPOINT ["/app/router"]
