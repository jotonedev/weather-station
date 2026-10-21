# syntax=docker/dockerfile:1
# Build context: system root (run from ./system)
# Usage:
#   docker build -f go/database/Dockerfile -t weather-station/database .

ARG HEALTHPROBE_IMAGE=weather-station/healthprobe:latest
FROM ${HEALTHPROBE_IMAGE} AS healthprobe

FROM --platform=$BUILDPLATFORM golang:1.27-bookworm AS build-stage

ARG TARGETOS=linux
ARG TARGETARCH

ENV GOWORK=off

RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc-aarch64-linux-gnu \
    g++-aarch64-linux-gnu \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /workspace

# Copy module manifests for dependency caching
COPY go/common/go.mod go/common/go.sum ./go/common/
COPY proto/gen/go/go.mod proto/gen/go/go.sum ./proto/gen/go/
COPY go/database/go.mod go/database/go.sum ./go/database/

# Download dependencies using BuildKit cache
RUN --mount=type=cache,target=/go/pkg/mod \
    (cd go/common && go mod download) && \
    (cd proto/gen/go && go mod download) && \
    (cd go/database && go mod download)

# Copy source code required for the build
COPY go/common/ ./go/common/
COPY proto/gen/go/ ./proto/gen/go/
COPY go/database/ ./go/database/

RUN mkdir -p /data

# Compile binary (CGO_ENABLED=1 for embedded DuckDB)
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CC=$([ "$TARGETARCH" = "arm64" ] && echo "aarch64-linux-gnu-gcc" || echo "gcc") \
    CXX=$([ "$TARGETARCH" = "arm64" ] && echo "aarch64-linux-gnu-g++" || echo "g++") \
    CGO_ENABLED=1 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -C go/database \
    -trimpath \
    -ldflags="-s -w" \
    -o /service \
    ./cmd

FROM gcr.io/distroless/cc-debian12:nonroot AS release-stage

ENV GO_ENV=production

WORKDIR /app

COPY --from=build-stage --chown=nonroot:nonroot /data /data
COPY --from=build-stage /service /app/service
COPY --from=healthprobe /grpc-health-probe /usr/local/bin/grpc-health-probe

EXPOSE 8090

HEALTHCHECK --interval=20s --timeout=3s --start-period=5s \
    CMD [ "/usr/local/bin/grpc-health-probe", "-addr=127.0.0.1:8090", "-service=ready" ]

ENTRYPOINT ["/app/service"]
