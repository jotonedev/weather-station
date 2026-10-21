# syntax=docker/dockerfile:1
# Build context: system root (run from ./system)
# Usage:
#   docker build -f go/test-generator/Dockerfile -t weather-station/test-generator .

ARG HEALTHPROBE_IMAGE=weather-station/healthprobe:latest
FROM ${HEALTHPROBE_IMAGE} AS healthprobe

FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS build-stage

ARG TARGETOS=linux
ARG TARGETARCH

ENV GOWORK=off

WORKDIR /workspace

# Copy module manifests for dependency caching
COPY go/common/go.mod go/common/go.sum ./go/common/
COPY go/test-generator/go.mod go/test-generator/go.sum ./go/test-generator/

# Download dependencies using BuildKit cache
RUN --mount=type=cache,target=/go/pkg/mod \
    (cd go/common && go mod download) && \
    (cd go/test-generator && go mod download)

# Copy source code required for the build
COPY go/common/ ./go/common/
COPY go/test-generator/ ./go/test-generator/

# Compile binary (pure Go, statically linked)
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -C go/test-generator \
    -trimpath \
    -ldflags="-s -w" \
    -o /service \
    ./cmd

FROM gcr.io/distroless/static-debian12:nonroot AS release-stage

ENV GO_ENV=production

WORKDIR /app

COPY --from=build-stage /service /app/service
COPY --from=healthprobe /grpc-health-probe /usr/local/bin/grpc-health-probe

EXPOSE 8090

HEALTHCHECK --interval=20s --timeout=3s --start-period=5s \
    CMD [ "/usr/local/bin/grpc-health-probe", "-addr=127.0.0.1:8090", "-service=ready" ]

ENTRYPOINT ["/app/service"]
