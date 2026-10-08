# syntax=docker/dockerfile:1
# Build context = repository root (run from ./system):
#   docker build -f go/frontend/Dockerfile -t weather-station/frontend .

FROM golang:1.27-alpine AS build-stage

WORKDIR /workspace

COPY go.work go.work.sum ./

COPY go/common/go.mod go/common/go.sum go/common/
COPY go/frontend/go.mod go/frontend/go.sum go/frontend/

RUN go env -w GOCACHE=/go-cache     && (cd go/common && go mod download)     && (cd go/frontend && go mod download)

COPY go/common/ go/common/
COPY go/frontend/ go/frontend/

RUN --mount=type=cache,target=/go-cache     CGO_ENABLED=0 go build -installsuffix 'static' -o /service ./go/frontend/cmd

FROM alpine/curl:8.17.0 AS release-stage

ARG GO_ENV="production"
ENV GO_ENV=${GO_ENV}

WORKDIR /app

COPY --from=build-stage /service /app/service

EXPOSE 8080
EXPOSE 8090

HEALTHCHECK --interval=20s --timeout=3s --start-period=5s CMD [ "/usr/bin/curl", "-f", "http://127.0.0.1:8080/health" ]

ENTRYPOINT ["/app/service"]
