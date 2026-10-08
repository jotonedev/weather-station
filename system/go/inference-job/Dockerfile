# syntax=docker/dockerfile:1
# Build context = repository root (run from ./system):
#   docker build -f go/inference-job/Dockerfile -t weather-station/inference-job .

FROM golang:1.27-alpine AS build-stage

WORKDIR /workspace

COPY go.work go.work.sum ./

COPY go/common/go.mod go/common/go.sum go/common/
COPY go/inference-job/go.mod go/inference-job/go.sum go/inference-job/

RUN go env -w GOCACHE=/go-cache     && (cd go/common && go mod download)     && (cd go/inference-job && go mod download)

COPY go/common/ go/common/
COPY go/inference-job/ go/inference-job/

RUN --mount=type=cache,target=/go-cache     CGO_ENABLED=0 go build -installsuffix 'static' -o /service ./go/inference-job/cmd

FROM alpine/curl:8.17.0 AS release-stage

ARG GO_ENV="production"
ENV GO_ENV=${GO_ENV}

WORKDIR /app

COPY --from=build-stage /service /app/service

EXPOSE 8080
EXPOSE 8090

HEALTHCHECK --interval=20s --timeout=3s --start-period=5s CMD [ "/usr/bin/curl", "-f", "http://127.0.0.1:8080/health" ]

ENTRYPOINT ["/app/service"]
