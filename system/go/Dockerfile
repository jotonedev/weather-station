# syntax=docker/dockerfile:1
# Build context: system root (run from ./system)
# Usage:
#   docker build -f go/Dockerfile --build-arg SERVICE=<service-name> -t weather-station/<service-name> .

ARG GO_VERSION=1.27
FROM golang:${GO_VERSION}-bookworm AS build-stage

ARG SERVICE
RUN test -n "$SERVICE" || (echo "SERVICE build-arg must be specified (e.g. data-ingest, database, frontend, inference-job)" && false)

WORKDIR /workspace

# Copy workspace definition and all module manifests for dependency caching
COPY go.work go.work.sum ./
COPY go/common/go.mod go/common/go.sum ./go/common/
COPY proto/gen/go/go.mod proto/gen/go/go.sum ./proto/gen/go/
COPY go/data-ingest/go.mod go/data-ingest/go.sum ./go/data-ingest/
COPY go/database/go.mod go/database/go.sum ./go/database/
COPY go/frontend/go.mod go/frontend/go.sum ./go/frontend/
COPY go/inference-job/go.mod go/inference-job/go.sum ./go/inference-job/

# Download dependencies using BuildKit cache
RUN --mount=type=cache,target=/go/pkg/mod \
    (cd go/common && go mod download) && \
    (cd proto/gen/go && go mod download) && \
    (cd go/${SERVICE} && go mod download)

# Copy source code required for the build
COPY go/common/ ./go/common/
COPY proto/gen/go/ ./proto/gen/go/
COPY go/${SERVICE}/ ./go/${SERVICE}/

# Compile binary (CGO_ENABLED=1 required for embedded DuckDB in database service)
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /service \
    ./go/${SERVICE}/cmd

# Compile static health probe utility
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /grpc-health-probe \
    ./go/common/cmd/healthprobe

FROM debian:bookworm-slim AS release-stage

ARG GO_ENV="production"
ENV GO_ENV=${GO_ENV}

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates libstdc++6 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

RUN mkdir -p /data && chown -R 65534:65534 /data

COPY --from=build-stage /service /app/service
COPY --from=build-stage /grpc-health-probe /usr/local/bin/grpc-health-probe

USER 65534:65534

EXPOSE 8080 8090

HEALTHCHECK --interval=20s --timeout=3s --start-period=5s \
    CMD [ "/usr/local/bin/grpc-health-probe", "-addr=127.0.0.1:8090", "-service=ready" ]

ENTRYPOINT ["/app/service"]
