# syntax=docker/dockerfile:1
# Build context: system root (run from ./system)
# Usage:
#   docker build -f go/common/cmd/healthprobe/Dockerfile -t weather-station/healthprobe .

FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS build-stage

ARG TARGETOS=linux
ARG TARGETARCH

ENV GOWORK=off

WORKDIR /workspace/go/common
COPY go/common/go.mod go/common/go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY go/common/ ./

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /grpc-health-probe \
    ./cmd/healthprobe

FROM scratch AS release-stage

COPY --from=build-stage /grpc-health-probe /grpc-health-probe

USER 65532:65532

ENTRYPOINT ["/grpc-health-probe"]
