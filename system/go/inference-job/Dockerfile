FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS build-stage

ARG TARGETOS=linux
ARG TARGETARCH

WORKDIR /workspace

# Copy workspace definition and required module manifests
COPY go.work go.work.sum ./
COPY go/common/go.mod go/common/go.sum ./go/common/
COPY proto/gen/go/go.mod proto/gen/go/go.sum ./proto/gen/go/
COPY go/inference-job/go.mod go/inference-job/go.sum ./go/inference-job/

# Download dependencies using BuildKit cache
RUN --mount=type=cache,target=/go/pkg/mod \
    (cd go/common && go mod download) && \
    (cd proto/gen/go && go mod download) && \
    (cd go/inference-job && go mod download)

# Copy source code required for the build
COPY go/common/ ./go/common/
COPY proto/gen/go/ ./proto/gen/go/
COPY go/inference-job/ ./go/inference-job/

# Compile binary (pure Go, statically linked)
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /service \
    ./go/inference-job/cmd

FROM gcr.io/distroless/static-debian12:nonroot AS release-stage

ENV GO_ENV=production

WORKDIR /app

COPY --from=build-stage /service /app/service

ENTRYPOINT ["/app/service"]
